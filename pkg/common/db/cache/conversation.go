package cache

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/relation"
	relationTb "github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/table/relation"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/utils"
	"github.com/dtm-labs/rockscache"
	"github.com/go-redis/redis/v8"
)

const (
	conversationKey                          = "CONVERSATION:"
	conversationIDsKey                       = "CONVERSATION_IDS:"
	recvMsgOptKey                            = "RECV_MSG_OPT:"
	superGroupRecvMsgNotNotifyUserIDsKey     = "SUPER_GROUP_RECV_MSG_NOT_NOTIFY_USER_IDS:"
	superGroupRecvMsgNotNotifyUserIDsHashKey = "SUPER_GROUP_RECV_MSG_NOT_NOTIFY_USER_IDS_HASH:"

	conversationExpireTime = time.Second * 60 * 60 * 12
)

// arg fn will exec when no data in cache
type ConversationCache interface {
	metaCache
	NewCache() ConversationCache
	// get user's conversationIDs from cache
	GetUserConversationIDs(ctx context.Context, ownerUserID string) ([]string, error)
	DelConversationIDs(userIDs []string) ConversationCache
	// get one conversation from cache
	GetConversation(ctx context.Context, ownerUserID, conversationID string) (*relationTb.ConversationModel, error)
	DelConvsersations(ownerUserID string, conversationIDs []string) ConversationCache
	DelUsersConversation(ownerUserIDs []string, conversationID string) ConversationCache
	// get one conversation from cache
	GetConversations(ctx context.Context, ownerUserID string, conversationIDs []string) ([]*relationTb.ConversationModel, error)
	// get one user's all conversations from cache
	GetUserAllConversations(ctx context.Context, ownerUserID string) ([]*relationTb.ConversationModel, error)
	// get user conversation recv msg from cache
	GetUserRecvMsgOpt(ctx context.Context, ownerUserID, conversationID string) (opt int, err error)
	DelUserRecvMsgOpt(ownerUserID, conversationID string) ConversationCache
	// get one super group recv msg but do not notification userID list
	GetSuperGroupRecvMsgNotNotifyUserIDs(ctx context.Context, groupID string) (userIDs []string, err error)
	DelSuperGroupRecvMsgNotNotifyUserIDs(groupID string) ConversationCache
	// get one super group recv msg but do not notification userID list hash
	GetSuperGroupRecvMsgNotNotifyUserIDsHash(ctx context.Context, groupID string) (hash uint64, err error)
	DelSuperGroupRecvMsgNotNotifyUserIDsHash(groupID string) ConversationCache
}

func NewConversationRedis(rdb redis.UniversalClient, opts rockscache.Options, db relationTb.ConversationModelInterface) ConversationCache {
	rcClient := rockscache.NewClient(rdb, opts)
	return &ConversationRedisCache{rcClient: rcClient, metaCache: NewMetaCacheRedis(rcClient), conversationDB: db, expireTime: conversationExpireTime}
}

type ConversationRedisCache struct {
	metaCache
	rcClient       *rockscache.Client
	conversationDB relationTb.ConversationModelInterface
	expireTime     time.Duration
}

func NewNewConversationRedis(rdb redis.UniversalClient, conversationDB *relation.ConversationGorm, options rockscache.Options) ConversationCache {
	rcClient := rockscache.NewClient(rdb, options)
	return &ConversationRedisCache{rcClient: rcClient, metaCache: NewMetaCacheRedis(rcClient), conversationDB: conversationDB, expireTime: conversationExpireTime}
}

func (c *ConversationRedisCache) NewCache() ConversationCache {
	return &ConversationRedisCache{rcClient: c.rcClient, metaCache: NewMetaCacheRedis(c.rcClient, c.metaCache.GetPreDelKeys()...), conversationDB: c.conversationDB, expireTime: c.expireTime}
}

func (c *ConversationRedisCache) getConversationKey(ownerUserID, conversationID string) string {
	return conversationKey + ownerUserID + ":" + conversationID
}

func (c *ConversationRedisCache) getConversationIDsKey(ownerUserID string) string {
	return conversationIDsKey + ownerUserID
}

func (c *ConversationRedisCache) getSuperGroupRecvNotNotifyUserIDsKey(groupID string) string {
	return superGroupRecvMsgNotNotifyUserIDsKey + groupID
}

func (c *ConversationRedisCache) getRecvMsgOptKey(ownerUserID, conversationID string) string {
	return recvMsgOptKey + ownerUserID + ":" + conversationID
}

func (c *ConversationRedisCache) getSuperGroupRecvNotNotifyUserIDsHashKey(groupID string) string {
	return superGroupRecvMsgNotNotifyUserIDsHashKey + groupID
}

func (c *ConversationRedisCache) GetUserConversationIDs(ctx context.Context, ownerUserID string) ([]string, error) {
	return getCache(ctx, c.rcClient, c.getConversationIDsKey(ownerUserID), c.expireTime, func(ctx context.Context) ([]string, error) {
		return c.conversationDB.FindUserIDAllConversationID(ctx, ownerUserID)
	})
}

func (c *ConversationRedisCache) DelConversationIDs(userIDs []string) ConversationCache {
	var keys []string
	for _, userID := range userIDs {
		keys = append(keys, c.getConversationIDsKey(userID))
	}
	cache := c.NewCache()
	cache.AddKeys(keys...)
	return cache
}

func (c *ConversationRedisCache) GetConversation(ctx context.Context, ownerUserID, conversationID string) (*relationTb.ConversationModel, error) {
	return getCache(ctx, c.rcClient, c.getConversationKey(ownerUserID, conversationID), c.expireTime, func(ctx context.Context) (*relationTb.ConversationModel, error) {
		return c.conversationDB.Take(ctx, ownerUserID, conversationID)
	})
}

func (c *ConversationRedisCache) DelConvsersations(ownerUserID string, convsersationIDs []string) ConversationCache {
	var keys []string
	for _, conversationID := range convsersationIDs {
		keys = append(keys, c.getConversationKey(ownerUserID, conversationID))
	}
	cache := c.NewCache()
	cache.AddKeys(keys...)
	return cache
}

func (c *ConversationRedisCache) GetConversationIndex(convsation *relationTb.ConversationModel, keys []string) (int, error) {
	key := c.getConversationKey(convsation.OwnerUserID, convsation.ConversationID)
	for _i, _key := range keys {
		if _key == key {
			return _i, nil
		}
	}
	return 0, errors.New("not found key:" + key + " in keys")
}

func (c *ConversationRedisCache) GetConversations(ctx context.Context, ownerUserID string, conversationIDs []string) ([]*relationTb.ConversationModel, error) {
	var keys []string
	for _, conversarionID := range conversationIDs {
		keys = append(keys, c.getConversationKey(ownerUserID, conversarionID))
	}
	return batchGetCache(ctx, c.rcClient, keys, c.expireTime, c.GetConversationIndex, func(ctx context.Context) ([]*relationTb.ConversationModel, error) {
		return c.conversationDB.Find(ctx, ownerUserID, conversationIDs)
	})
}

func (c *ConversationRedisCache) GetUserAllConversations(ctx context.Context, ownerUserID string) ([]*relationTb.ConversationModel, error) {
	conversationIDs, err := c.GetUserConversationIDs(ctx, ownerUserID)
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, conversarionID := range conversationIDs {
		keys = append(keys, c.getConversationKey(ownerUserID, conversarionID))
	}
	return batchGetCache(ctx, c.rcClient, keys, c.expireTime, c.GetConversationIndex, func(ctx context.Context) ([]*relationTb.ConversationModel, error) {
		return c.conversationDB.FindUserIDAllConversations(ctx, ownerUserID)
	})
}

func (c *ConversationRedisCache) GetUserRecvMsgOpt(ctx context.Context, ownerUserID, conversationID string) (opt int, err error) {
	return getCache(ctx, c.rcClient, c.getRecvMsgOptKey(ownerUserID, conversationID), c.expireTime, func(ctx context.Context) (opt int, err error) {
		return c.conversationDB.GetUserRecvMsgOpt(ctx, ownerUserID, conversationID)
	})
}

func (c *ConversationRedisCache) GetSuperGroupRecvMsgNotNotifyUserIDs(ctx context.Context, groupID string) (userIDs []string, err error) {
	return getCache(ctx, c.rcClient, c.getSuperGroupRecvNotNotifyUserIDsKey(groupID), c.expireTime, func(ctx context.Context) (userIDs []string, err error) {
		return c.conversationDB.FindSuperGroupRecvMsgNotNotifyUserIDs(ctx, groupID)
	})
}

func (c *ConversationRedisCache) DelUsersConversation(ownerUserIDs []string, conversationID string) ConversationCache {
	var keys []string
	for _, ownerUserID := range ownerUserIDs {
		keys = append(keys, c.getConversationKey(ownerUserID, conversationID))
	}
	cache := c.NewCache()
	cache.AddKeys(keys...)
	return cache
}

func (c *ConversationRedisCache) DelUserRecvMsgOpt(ownerUserID, conversationID string) ConversationCache {
	cache := c.NewCache()
	cache.AddKeys(c.getRecvMsgOptKey(ownerUserID, conversationID))
	return cache
}

func (c *ConversationRedisCache) DelSuperGroupRecvMsgNotNotifyUserIDs(groupID string) ConversationCache {
	cache := c.NewCache()
	cache.AddKeys(c.getSuperGroupRecvNotNotifyUserIDsKey(groupID))
	return cache
}

func (c *ConversationRedisCache) GetSuperGroupRecvMsgNotNotifyUserIDsHash(ctx context.Context, groupID string) (hash uint64, err error) {
	return getCache(ctx, c.rcClient, c.getSuperGroupRecvNotNotifyUserIDsHashKey(groupID), c.expireTime, func(ctx context.Context) (hash uint64, err error) {
		userIDs, err := c.GetSuperGroupRecvMsgNotNotifyUserIDs(ctx, groupID)
		if err != nil {
			return 0, err
		}
		utils.Sort(userIDs, true)
		bi := big.NewInt(0)
		bi.SetString(utils.Md5(strings.Join(userIDs, ";"))[0:8], 16)
		return bi.Uint64(), nil
	})
}

func (c *ConversationRedisCache) DelSuperGroupRecvMsgNotNotifyUserIDsHash(groupID string) ConversationCache {
	cache := c.NewCache()
	cache.AddKeys(c.getSuperGroupRecvNotNotifyUserIDsHashKey(groupID))
	return cache
}