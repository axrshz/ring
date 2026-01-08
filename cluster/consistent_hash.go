package cluster

type ConsistentHash struct {
    mu           sync.RWMutex
    hashRing     []uint32
    nodeMap      map[uint32]string
    virtualNodes int
}
```

**What are these fields?**

### 1. `hashRing []uint32`
This is a **sorted array of hash values** representing positions on a "ring".

Think of it as a circular clock face with numbers:
```
        0
    359   1
  358       2
 357         3
  ...       ...
    181   179
       180