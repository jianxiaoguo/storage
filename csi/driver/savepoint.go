package driver

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/drycc/storage/csi/provider"
	bolt "go.etcd.io/bbolt"
)

type SaveEntry struct {
	Point  provider.MountPoint  `json:"point"`
	Bucket provider.MountBucket `json:"bucket"`
}

type Savepoint struct {
	path    string
	lock    sync.RWMutex
	bucket  string
	timeout time.Duration
}

func NewSavepoint(path string) (*Savepoint, error) {
	bucket := "savepoint"
	timeout := 10 * time.Second
	if db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: timeout}); err != nil {
		return nil, err
	} else {
		defer db.Close()
		db.Update(func(tx *bolt.Tx) error {
			if b := tx.Bucket([]byte(bucket)); b == nil {
				if _, err := tx.CreateBucket([]byte(bucket)); err != nil {
					return err
				}
			}
			return nil
		})
	}
	return &Savepoint{path: path, bucket: bucket, timeout: timeout}, nil
}

func (p *Savepoint) View(handle func(saveEntry *SaveEntry)) error {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if db, err := bolt.Open(p.path, 0600, &bolt.Options{Timeout: p.timeout}); err != nil {
		return err
	} else {
		defer db.Close()
		db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(p.bucket))
			c := b.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				var saveEntry SaveEntry
				if err := json.Unmarshal(v, &saveEntry); err != nil {
					return err
				}
				handle(&saveEntry)
			}
			return nil
		})
	}
	return nil
}

func (p *Savepoint) Save(targetPath string, mountPoint *provider.MountPoint, mountBucket *provider.MountBucket) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if data, err := json.Marshal(SaveEntry{Point: *mountPoint, Bucket: *mountBucket}); err != nil {
		return err
	} else {
		if db, err := bolt.Open(p.path, 0600, &bolt.Options{Timeout: p.timeout}); err != nil {
			return err
		} else {
			defer db.Close()
			db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte(p.bucket))
				err := b.Put([]byte(targetPath), data)
				return err
			})
		}
	}
	return nil
}

func (p *Savepoint) Delete(volumeID string) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if db, err := bolt.Open(p.path, 0600, &bolt.Options{Timeout: p.timeout}); err != nil {
		return err
	} else {
		defer db.Close()
		db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(p.bucket))
			b.Delete([]byte(volumeID))
			return err
		})
	}
	return nil
}
