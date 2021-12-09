package opensearch

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	e "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
	"github.com/opensearch-project/opensearch-go/opensearchutil"
)

var (
	OpensearchIndexPrefix string = func() string {
		prefix := os.Getenv("OPENSEARCH_INDEX_PREFIX")
		if len(prefix) > 0 {
			return prefix
		} else {
			return os.Getenv("DOPPLER_CONFIG")
		}
	}()
	OpensearchDomain   string = os.Getenv("OPENSEARCH_DOMAIN")
	OpensearchPassword string = os.Getenv("OPENSEARCH_PASSWORD")
	OpensearchUsername string = os.Getenv("OPENSEARCH_USERNAME")
)

type Index string

var (
	IndexSessions Index = "sessions"
	IndexFields   Index = "fields"
	IndexErrors   Index = "errors"
)

func GetIndex(suffix Index) string {
	return OpensearchIndexPrefix + "_" + string(suffix)
}

type Client struct {
	Client        *opensearch.Client
	BulkIndexer   opensearchutil.BulkIndexer
	isInitialized bool
}

func NewOpensearchClient() (*Client, error) {
	client, err := opensearch.NewClient(opensearch.Config{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Addresses: []string{OpensearchDomain},
		Username:  OpensearchUsername,
		Password:  OpensearchPassword,
	})
	if err != nil {
		return nil, e.Wrap(err, "OPENSEARCH_ERROR failed to initialize opensearch client")
	}

	indexer, err := opensearchutil.NewBulkIndexer(opensearchutil.BulkIndexerConfig{
		Client:        client,
		NumWorkers:    4,                // The number of workers. Defaults to runtime.NumCPU().
		FlushBytes:    5e+6,             // The flush threshold in bytes. Defaults to 5MB.
		FlushInterval: 10 * time.Second, // The flush threshold as duration. Defaults to 30sec.
		OnError: func(ctx context.Context, err error) {
			log.Error(e.Wrap(err, "OPENSEARCH_ERROR bulk indexer error"))
		},
	})
	if err != nil {
		return nil, e.Wrap(err, "OPENSEARCH_ERROR failed to initialize opensearch bulk indexer")
	}

	return &Client{
		Client:        client,
		BulkIndexer:   indexer,
		isInitialized: true,
	}, nil
}

func (c *Client) Update(index Index, id int, obj map[string]interface{}) error {
	if c == nil || !c.isInitialized {
		return nil
	}

	documentId := strconv.Itoa(id)

	b, err := json.Marshal(obj)
	if err != nil {
		return e.Wrap(err, "OPENSEARCH_ERROR error marshalling map for update")
	}
	body := strings.NewReader(fmt.Sprintf("{ \"doc\" : %s }", string(b)))

	indexStr := GetIndex(index)

	item := opensearchutil.BulkIndexerItem{
		Index:      indexStr,
		Action:     "update",
		DocumentID: documentId,
		Body:       body,
		OnSuccess: func(ctx context.Context, item opensearchutil.BulkIndexerItem, res opensearchutil.BulkIndexerResponseItem) {
			log.Infof("OPENSEARCH_SUCCESS (%s : %s) [%d] %s", indexStr, item.DocumentID, res.Status, res.Result)
		},
		OnFailure: func(ctx context.Context, item opensearchutil.BulkIndexerItem, res opensearchutil.BulkIndexerResponseItem, err error) {
			if err != nil {
				log.Errorf("OPENSEARCH_ERROR (%s : %s) %s", indexStr, item.DocumentID, err)
			} else {
				log.Errorf("OPENSEARCH_ERROR (%s : %s) %s %s", indexStr, item.DocumentID, res.Error.Type, res.Error.Reason)
			}
		},
	}

	if err := c.BulkIndexer.Add(context.Background(), item); err != nil {
		return e.Wrap(err, "OPENSEARCH_ERROR error adding bulk indexer item for update")
	}

	return nil
}

func (c *Client) Index(index Index, id int, obj interface{}) error {
	if c == nil || !c.isInitialized {
		return nil
	}

	documentId := strconv.Itoa(id)

	b, err := json.Marshal(obj)
	if err != nil {
		return e.Wrap(err, "OPENSEARCH_ERROR error marshalling map for index")
	}
	body := strings.NewReader(string(b))

	indexStr := GetIndex(index)

	item := opensearchutil.BulkIndexerItem{
		Index:      indexStr,
		Action:     "index",
		DocumentID: documentId,
		Body:       body,
		OnSuccess: func(ctx context.Context, item opensearchutil.BulkIndexerItem, res opensearchutil.BulkIndexerResponseItem) {
			log.Infof("OPENSEARCH_SUCCESS (%s : %s) [%d] %s", indexStr, item.DocumentID, res.Status, res.Result)
		},
		OnFailure: func(ctx context.Context, item opensearchutil.BulkIndexerItem, res opensearchutil.BulkIndexerResponseItem, err error) {
			if err != nil {
				log.Errorf("OPENSEARCH_ERROR (%s : %s) %s", indexStr, item.DocumentID, err)
			} else {
				log.Errorf("OPENSEARCH_ERROR (%s : %s) %s %s", indexStr, item.DocumentID, res.Error.Type, res.Error.Reason)
			}
		},
	}

	if err := c.BulkIndexer.Add(context.Background(), item); err != nil {
		return e.Wrap(err, "OPENSEARCH_ERROR error adding bulk indexer item for index")
	}

	return nil
}

func (c *Client) AppendToField(index Index, sessionID int, fieldName string, fields []interface{}) error {
	if c == nil || !c.isInitialized {
		return nil
	}

	// Nothing to append, skip the OpenSearch request
	if len(fields) == 0 {
		return nil
	}

	documentId := strconv.Itoa(sessionID)

	b, err := json.Marshal(fields)
	if err != nil {
		return e.Wrap(err, "OPENSEARCH_ERROR error marshalling fields")
	}
	body := strings.NewReader(fmt.Sprintf(`{"script" : {"source": "ctx._source.%s.addAll(params.toAppend)","params" : {"toAppend" : %s}}}`, fieldName, string(b)))

	indexStr := GetIndex(index)

	item := opensearchutil.BulkIndexerItem{
		Index:      indexStr,
		Action:     "update",
		DocumentID: documentId,
		Body:       body,
		OnSuccess: func(ctx context.Context, item opensearchutil.BulkIndexerItem, res opensearchutil.BulkIndexerResponseItem) {
			log.Infof("OPENSEARCH_SUCCESS (%s : %s) [%d] %s", indexStr, item.DocumentID, res.Status, res.Result)
		},
		OnFailure: func(ctx context.Context, item opensearchutil.BulkIndexerItem, res opensearchutil.BulkIndexerResponseItem, err error) {
			if err != nil {
				log.Errorf("OPENSEARCH_ERROR (%s : %s) %s", indexStr, item.DocumentID, err)
			} else {
				log.Errorf("OPENSEARCH_ERROR (%s : %s) %s %s", indexStr, item.DocumentID, res.Error.Type, res.Error.Reason)
			}
		},
	}

	if err := c.BulkIndexer.Add(context.Background(), item); err != nil {
		return e.Wrap(err, "OPENSEARCH_ERROR error adding bulk indexer item for update (append session fields)")
	}

	return nil

}

func (c *Client) IndexSynchronous(index Index, id int, obj interface{}) error {
	if c == nil || !c.isInitialized {
		return nil
	}

	documentId := strconv.Itoa(id)

	b, err := json.Marshal(obj)
	if err != nil {
		return e.Wrap(err, "OPENSEARCH_ERROR error marshalling map for index")
	}
	body := strings.NewReader(string(b))

	indexStr := GetIndex(index)

	req := opensearchapi.IndexRequest{
		Index:      indexStr,
		DocumentID: documentId,
		Body:       body,
	}

	res, err := req.Do(context.Background(), c.Client)
	if err != nil {
		return e.Wrap(err, "OPENSEARCH_ERROR error indexing document")
	}

	log.Infof("OPENSEARCH_SUCCESS (%s : %s) [%d] created", indexStr, documentId, res.StatusCode)

	return nil
}

func (c *Client) Close() error {
	if c == nil || !c.isInitialized {
		return nil
	}

	return c.BulkIndexer.Close(context.Background())
}