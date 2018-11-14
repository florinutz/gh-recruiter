package cache

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/birkelund/boltdbcache"

	"github.com/pkg/errors"

	"github.com/gregjones/httpcache"
)

type Cache struct {
	httpcache.Cache
	validity time.Duration
}

func NewCache(bucketName string, validity time.Duration) (cache *Cache, err error) {
	if cacheDir, err := os.UserCacheDir(); err != nil {
		return nil, err
	} else {
		c, err := boltdbcache.New(filepath.Join(cacheDir, bucketName))
		if err != nil {
			return nil, err
		}
		cache = &Cache{validity: validity, Cache: c}
	}
	return
}

type payload struct {
	CreationTime time.Time
	Query        interface{}
}

func (cache Cache) WriteQuery(q interface{}, variables map[string]interface{}) error {
	hash, err := getHashForCall(q, variables)
	if err != nil {
		return errors.Wrap(err, "coultn't compute ghv4 call hash")
	}

	cacheKey := fmt.Sprintf("query-%s", hash)

	gob.Register(q)

	payload := payload{CreationTime: time.Now(), Query: q}

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	if err = encoder.Encode(payload); err != nil {
		return errors.Wrap(err, "cache data encoding error")
	}
	fmt.Println(buf.String())

	cache.Set(cacheKey, buf.Bytes())

	return nil
}

func (cache Cache) ReadQuery(q interface{}, variables map[string]interface{}) (interface{}, error) {
	hash, err := getHashForCall(q, variables)
	if err != nil {
		return nil, err
	}
	cacheKey := fmt.Sprintf("query-%s", hash)

	item, ok := cache.Get(cacheKey)
	if !ok {
		return nil, fmt.Errorf("no cache for key %s", cacheKey)
	}

	buf := bytes.NewBuffer(item)
	fmt.Println(buf.String())

	decoder := gob.NewDecoder(buf)
	empty := payload{}
	if err = decoder.Decode(&empty); err != nil {
		return nil, errors.Wrap(err, "cache unmarshaling error")
	}

	if time.Since(empty.CreationTime) > cache.validity {
		return nil, fmt.Errorf("cache expired for key %s", cacheKey)
	}

	return empty.Query, nil
}

// getHashForCall returns a hash for a specific query - variables combination
func getHashForCall(q interface{}, variables map[string]interface{}) (string, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(q); err != nil {
		return "", errors.Wrap(err, "error while gob encoding query")
	}

	if err := enc.Encode(variables); err != nil {
		return "", errors.Wrap(err, "error while gob encoding vars")
	}

	sum := md5.Sum(buf.Bytes())

	return hex.EncodeToString(sum[:]), nil
}
