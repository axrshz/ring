// client/client.go
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"ring/cluster"
)

type Client struct {
    cluster *cluster.Cluster
    client  *http.Client
}

func NewClient(nodes []string) *Client {
    c := &Client{
        cluster: cluster.NewCluster(),
        client:  &http.Client{},
    }

    for _, node := range nodes {
        c.cluster.AddNode(node)
    }

    return c
}

func (c *Client) Set(key, value string, ttl int) error {
    node := c.cluster.GetNodeForKey(key)
    if node == "" {
        return fmt.Errorf("no nodes available")
    }

    reqBody, _ := json.Marshal(map[string]interface{}{
        "key":   key,
        "value": value,
        "ttl":   ttl,
    })

    resp, err := c.client.Post(
        fmt.Sprintf("http://%s/set", node),
        "application/json",
        bytes.NewBuffer(reqBody),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("set failed: %d", resp.StatusCode)
    }

    return nil
}

func (c *Client) Get(key string) (string, error) {
    node := c.cluster.GetNodeForKey(key)
    if node == "" {
        return "", fmt.Errorf("no nodes available")
    }

    resp, err := c.client.Get(fmt.Sprintf("http://%s/get?key=%s", node, key))
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusNotFound {
        return "", fmt.Errorf("key not found")
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    return string(body), nil
}

func (c *Client) Delete(key string) error {
    node := c.cluster.GetNodeForKey(key)
    if node == "" {
        return fmt.Errorf("no nodes available")
    }

    req, _ := http.NewRequest("DELETE", fmt.Sprintf("http://%s/delete?key=%s", node, key), nil)
    resp, err := c.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    return nil
}