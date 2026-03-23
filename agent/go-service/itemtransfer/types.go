package itemtransfer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

type itemOrderData struct {
	Items         map[string]itemInfo `json:"items"`
	CategoryOrder map[string][]string `json:"category_order"`
}

type itemInfo struct {
	Name     string `json:"name"`
	Category string `json:"category"`
}

type fallbackParams struct {
	TargetClass int    `json:"target_class"`
	Descending  bool   `json:"descending"`
	Side        string `json:"side"`
}

type gridItem struct {
	Box     [4]int
	ClassID uint64
	Score   float64
	CenterX int
	CenterY int
}

const (
	componentName = "itemtransfer"

	repoNNDNode    = "ItemTransferDetectAllItems"
	bagNNDNode     = "ItemTransferDetectAllItemsBag"
	tooltipOCRNode = "ItemTransferTooltipOCR"

	tooltipOffsetX = 31
	tooltipOffsetY = 6
	tooltipWidth   = 117
	tooltipHeight  = 58
)

var (
	cachedData     *itemOrderData
	cachedDataOnce sync.Once
	cachedDataErr  error

	resourcePath atomic.Value // string – set by resourcePathSink
)

type resourcePathSink struct{}

var _ maa.ResourceEventSink = &resourcePathSink{}

func (c *resourcePathSink) OnResourceLoading(_ *maa.Resource, status maa.EventStatus, detail maa.ResourceLoadingDetail) {
	if status != maa.EventStatusSucceeded || detail.Path == "" {
		return
	}
	abs := detail.Path
	if p, err := filepath.Abs(detail.Path); err == nil {
		abs = p
	}
	resourcePath.Store(abs)
	log.Debug().Str("resource_path", abs).Msg("resource loaded; cached path for itemtransfer")
}

func getResourceBase() string {
	if v := resourcePath.Load(); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func loadItemOrderData() (*itemOrderData, error) {
	cachedDataOnce.Do(func() {
		dir, err := findDataDir()
		if err != nil {
			cachedDataErr = err
			return
		}
		b, err := os.ReadFile(filepath.Join(dir, "item_order.json"))
		if err != nil {
			cachedDataErr = err
			return
		}
		var data itemOrderData
		if err := json.Unmarshal(b, &data); err != nil {
			cachedDataErr = err
			return
		}
		cachedData = &data
		log.Info().
			Str("component", componentName).
			Int("item_count", len(data.Items)).
			Int("category_count", len(data.CategoryOrder)).
			Msg("item order data loaded")
	})
	return cachedData, cachedDataErr
}

func findDataDir() (string, error) {
	const target = "item_order.json"

	relPaths := []string{
		filepath.Join("assets", "data", "ItemTransfer"),
		filepath.Join("data", "ItemTransfer"),
	}

	var tried []string

	check := func(base string) (string, bool) {
		for _, rel := range relPaths {
			cand := filepath.Join(base, rel)
			if fileExists(filepath.Join(cand, target)) {
				return cand, true
			}
			tried = append(tried, cand)
		}
		return "", false
	}

	// 1. Environment variable override.
	if v := strings.TrimSpace(os.Getenv("MAAEND_ITEMTRANSFER_DATA_DIR")); v != "" {
		if fileExists(filepath.Join(v, target)) {
			return v, nil
		}
		tried = append(tried, v)
	}

	// 2. Walk up from cwd.
	if wd, err := os.Getwd(); err == nil {
		base := wd
		for i := 0; i < 8; i++ {
			if dir, ok := check(base); ok {
				return dir, nil
			}
			parent := filepath.Dir(base)
			if parent == base {
				break
			}
			base = parent
		}
	}

	// 3. Walk up from executable path.
	if exePath, err := os.Executable(); err == nil {
		base := filepath.Dir(exePath)
		for i := 0; i < 8; i++ {
			if dir, ok := check(base); ok {
				return dir, nil
			}
			parent := filepath.Dir(base)
			if parent == base {
				break
			}
			base = parent
		}
	}

	// 4. Fallback: MaaFramework resource path (e.g. assets/resource); try parent & grandparent.
	if base := getResourceBase(); base != "" {
		base = filepath.Clean(base)
		for _, b := range []string{filepath.Dir(base), filepath.Dir(filepath.Dir(base))} {
			if dir, ok := check(b); ok {
				return dir, nil
			}
		}
	}

	return "", fmt.Errorf("cannot find %s in any of %d candidate paths: %v; set MAAEND_ITEMTRANSFER_DATA_DIR", target, len(tried), tried)
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// inferSide returns the side from params if set, otherwise infers from the
// pipeline node name: nodes containing "Bag" operate on the bag area.
func inferSide(paramSide, taskName string) string {
	if paramSide != "" {
		return paramSide
	}
	if strings.Contains(taskName, "Bag") {
		return "bag"
	}
	return "repo"
}
