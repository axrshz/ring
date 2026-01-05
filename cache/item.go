// cache/item.go
package cache

import "time"

type Item struct {
    Value      []byte
    Expiration int64 // Unix timestamp
}

func (i *Item) IsExpired() bool {
    if i.Expiration == 0 {
        return false
    }
    return time.Now().UnixNano() > i.Expiration
}