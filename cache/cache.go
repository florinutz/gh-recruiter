package cache

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

type cachePayload struct {
	creationTime time.Time
	query        interface{}
}

func (cache Cache) WriteQuery(q interface{}, variables map[string]interface{}) error {
	hash, err := getHashForCall(q, variables)
	if err != nil {
		return errors.Wrap(err, "coultn't compute ghv4 call hash")
	}
	cacheKey := fmt.Sprintf("query-%s", hash)
	toMarshal := cachePayload{creationTime: time.Now(), query: q}

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	encoder.Encode(toMarshal)

	cache.Set(cacheKey, buf.Bytes())

	return nil
}

func (cache Cache) ReadQuery(q interface{}, variables map[string]interface{}) (
	interface{}, error) {
	hash, err := getHashForCall(q, variables)
	if err != nil {
		return nil, err
	}
	cacheKey := fmt.Sprintf("query-%s", hash)

	wt := cachePayload{}
	item, ok := cache.Get(cacheKey)
	if !ok {
		return nil, fmt.Errorf("no cache for key %s", cacheKey)
	}

	buf := bytes.NewBuffer(item)

	decoder := gob.NewDecoder(buf)
	err = decoder.Decode(&wt)
	if err != nil {
		return nil, errors.Wrap(err, "cache unmarshaling error")
	}

	if time.Since(wt.creationTime) > cache.validity {
		return nil, fmt.Errorf("cache expired for key %s", cacheKey)
	}

	return wt.query, nil
}

func getJson(v interface{}, indent string, forZeroVal bool) (string, error) {
	if v == nil {
		return "", errors.New("nil input")
	}

	if forZeroVal {
		// make v its 0 value so we get consistent hashes even for incoming values
		ptr := reflect.New(reflect.TypeOf(v))
		v = ptr.Elem().Interface()
	}

	var buf bytes.Buffer

	w := json.NewEncoder(&buf)
	w.SetIndent("", indent)

	err := w.Encode(v)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func getHashForCall(q interface{}, variables map[string]interface{}) (string, error) {
	jsonQuery, err := getJson(q, "", true)
	if err != nil {
		return "", err
	}
	jsonVars, err := getJson(variables, "", false)
	if err != nil {
		return "", err
	}

	sum := md5.Sum([]byte(jsonQuery + jsonVars))

	return hex.EncodeToString(sum[:]), nil
}
