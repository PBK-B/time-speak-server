package hashtag

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"regexp"
	"time"
	"time_speak_server/src/exception"
	"time_speak_server/src/opts"
	"time_speak_server/src/service/cache"
	"time_speak_server/src/service/user"
)

type Svc struct {
	Config
	redis *redis.Client
	c     *cache.Svc
	m     *mongo.Collection
}

func NewHashTagSvc(conf Config, db *mongo.Database, redis *redis.Client) *Svc {
	return &Svc{
		Config: conf,
		redis:  redis,
		m:      db.Collection("hashtag"),
		c:      cache.NewCacheSvc(redis),
	}
}

func (s *Svc) NewHashTag(ctx context.Context, name string) (primitive.ObjectID, error) {
	id, err := user.GetUserFromJwt(ctx)
	if err != nil {
		return primitive.NilObjectID, err
	}
	hashtag := HashTag{
		ObjectID:   primitive.NewObjectID(),
		Uid:        id,
		Name:       name,
		Archived:   false,
		CreateTime: time.Now().Unix(),
	}
	_, err = s.m.InsertOne(ctx, hashtag)
	return hashtag.ObjectID, err
}

// UpdateHashTag 更新标签
func (s *Svc) UpdateHashTag(ctx context.Context, id primitive.ObjectID, opts ...opts.Option) error {
	uid, err := user.GetUserFromJwt(ctx)
	if err != nil {
		return err
	}
	toUpdate := bson.M{"update_time": time.Now().Unix()}
	for _, f := range opts {
		toUpdate = f(toUpdate)
	}
	_, err = s.m.UpdateOne(ctx, bson.M{"uid": uid, "_id": id}, bson.M{"$set": toUpdate})
	s.c.Del(ctx, fmt.Sprintf("#-%s", id.Hex()))
	return err
}

// DeleteHashTag 删除标签
func (s *Svc) DeleteHashTag(ctx context.Context, id primitive.ObjectID) error {
	uid, err := user.GetUserFromJwt(ctx)
	if err != nil {
		return err
	}
	result, err := s.m.DeleteOne(ctx, bson.M{"_id": id, "uid": uid, "archived": true}) // 只有归档的才能删除
	s.c.Del(ctx, fmt.Sprintf("#-%s", id.Hex()))
	if result.DeletedCount == 0 {
		return exception.ErrHashTagNotFound
	}
	return err
}

func (s *Svc) GetOrInsertHashTag(ctx context.Context, name string) (primitive.ObjectID, error) {
	uid, err := user.GetUserFromJwt(ctx)
	if err != nil {
		return primitive.NilObjectID, err
	}
	var tag HashTag
	err = s.m.FindOne(ctx, bson.M{"uid": uid, "name": name}).Decode(&tag)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			hashtag, err := s.NewHashTag(ctx, name)
			if err != nil {
				return primitive.NilObjectID, err
			}
			return hashtag, nil
		}
		return primitive.NilObjectID, err
	}
	return tag.ObjectID, nil
}

func (s *Svc) GetHashTag(ctx context.Context, name string) (*HashTag, error) {
	uid, err := user.GetUserFromJwt(ctx)
	if err != nil {
		return nil, err
	}
	f := func() ([]byte, error) {
		var tag HashTag
		err = s.m.FindOne(ctx, bson.M{"uid": uid, "name": name}).Decode(&tag)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, exception.ErrHashTagNotFound
			}
			return nil, err
		}
		return bson.Marshal(tag)
	}
	var tag HashTag
	// Redis 缓存
	result, err := s.c.Get(ctx, fmt.Sprintf("Tag-%s", name), time.Minute*time.Duration(10), f)
	if err != nil {
		return nil, err
	}
	err = bson.Unmarshal(result, &tag)
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// GetHashTags 获取标签列表
func (s *Svc) GetHashTags(ctx context.Context, page, size int64, byCreate, desc, archived bool) ([]*HashTag, error) {
	uid, err := user.GetUserFromJwt(ctx)
	if err != nil {
		return nil, err
	}
	var tags []*HashTag
	skip := page * size
	order := 1
	if desc {
		order = -1
	}
	sort := "update_time"
	if byCreate {
		sort = "create_time"
	}
	cur, err := s.m.Find(ctx, bson.M{"uid": uid, "archived": archived}, &options.FindOptions{
		Skip:  &skip,
		Limit: &size,
		Sort:  bson.M{sort: order},
	}) // 只能获取自己的标签
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var tag HashTag
		err := cur.Decode(&tag)
		if err != nil {
			return nil, err
		}
		tags = append(tags, &tag)
	}
	return tags, nil
}

func (s *Svc) GetHashTagByID(ctx context.Context, id primitive.ObjectID) (*HashTag, error) {
	uid, err := user.GetUserFromJwt(ctx)
	if err != nil {
		return nil, err
	}
	f := func() ([]byte, error) {
		var tag HashTag
		err := s.m.FindOne(ctx, bson.M{"uid": uid, "_id": id}).Decode(&tag) // 只能获取自己的标签
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, exception.ErrHashTagNotFound
			}
			return nil, err
		}
		return bson.Marshal(tag)
	}
	var tag HashTag
	// Redis 缓存
	result, err := s.c.Get(ctx, fmt.Sprintf("#-%s", id.Hex()), time.Minute*time.Duration(10), f)
	if err != nil {
		return nil, err
	}
	err = bson.Unmarshal(result, &tag)
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (s *Svc) MakeHashTags(ctx context.Context, content string) ([]primitive.ObjectID, error) {
	tags := ParseHashTags(content)
	var ids []primitive.ObjectID
	for _, tag := range tags {
		f := func() ([]byte, error) {
			hashtag, err := s.GetOrInsertHashTag(ctx, tag)
			if err != nil {
				return nil, err
			}
			return []byte(hashtag.Hex()), nil
		}
		// Redis 缓存
		result, err := s.c.Get(ctx, fmt.Sprintf("Tag-%s", tag), time.Minute*time.Duration(10), f)
		if err != nil {
			return nil, err
		}
		id, err := primitive.ObjectIDFromHex(string(result))
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func ParseHashTags(content string) []string {
	r, _ := regexp.Compile("#\\((\\S[^\\n]*?)\\)\\s|#(\\S[^\\n]*?)\\s")
	all := r.FindAllStringSubmatch(content+" ", -1) // 最后加一个空格，防止最后一个标签没有空格
	var tags []string
	for _, v := range all {
		tag := v[1]
		if len(tag) == 0 {
			tag = v[2]
		}
		tags = append(tags, tag)
	}
	return tags
}

/// #话题    #话题2    #(话题)
