package upstream

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"owas-api/internal/domain"
	"owas-api/internal/protocol/crypto"
	"owas-api/internal/protocol/protostuff"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newInMemoryHTTPClient(h http.HandlerFunc) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			// Normalize headers on a cloned request the way a real HTTP server does on read,
			// while verifying the client set wire headers cleanly.
			cloned := req.Clone(req.Context())
			canonical := make(http.Header, len(req.Header))
			for k, vals := range req.Header {
				for _, v := range vals {
					canonical.Add(k, v)
				}
			}
			cloned.Header = canonical
			rec := httptest.NewRecorder()
			h(rec, cloned)
			return rec.Result(), nil
		}),
	}
}

func TestStoreClientSearchAndSignatureVerification(t *testing.T) {
	httpClient := newInMemoryHTTPClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/watch/search/list" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Verify signature matches incoming path + raw query.
		expectedSign := crypto.ComputeAppStoreSign(
			r.URL.EscapedPath(),
			r.URL.RawQuery,
			r.Header.Get("User-Agent"),
			r.Header.Get("t"),
			r.Header.Get("id"),
		)
		if got := r.Header.Get("sign"); got != expectedSign {
			t.Errorf("signature mismatch: got %s, expected %s", got, expectedSign)
		}
		if r.Header.Get("accept") != "application/x2-protostuff" {
			t.Errorf("unexpected accept header: %s", r.Header.Get("accept"))
		}

		resBytes := protostuff.NewEncoder().
			AppendString(protostuff.PolyClassTag, "com.heytap.appstore.watch.domain.ResourceDto").
			AppendInt64(1, 31617863).
			AppendInt64(2, 25869985).
			AppendString(3, "微信").
			AppendString(4, "com.tencent.watchwechat").
			AppendString(5, "1.0.7").
			AppendInt64(6, 65543).
			AppendInt32(21, 12).
			Bytes()
		payload := protostuff.NewEncoder().
			AppendMessage(1, resBytes).
			AppendInt32(3, 0).
			Bytes()
		w.Header().Set("Content-Type", "application/x2-protostuff;charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	})

	client := NewStoreClient("https://mock.store.local", 5*time.Second, httpClient)
	got, err := client.Search(context.Background(), domain.DefaultDeviceProfile(), domain.SearchParams{
		Word:       "微信",
		SearchType: 1,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(got.Apps) != 1 || got.Apps[0].AppID != 31617863 || got.Apps[0].PkgName != "com.tencent.watchwechat" {
		t.Fatalf("unexpected search result: %+v", got)
	}
	if len(got.UnknownFields) != 0 || len(got.Apps[0].UnknownFields) != 0 {
		t.Fatalf("unexpected unknown fields: %+v", got)
	}
}

func TestStoreClientPolymorphicHomeCards(t *testing.T) {
	httpClient := newInMemoryHTTPClient(func(w http.ResponseWriter, r *http.Request) {
		banner := protostuff.NewEncoder().
			AppendInt64(1, 10).
			AppendString(2, "https://img/b.png").
			AppendString(5, "oap://mk/cardstyle?pk=600087").
			AppendString(6, "Top Banner").
			Bytes()
		navCard := protostuff.NewEncoder().
			AppendString(protostuff.PolyClassTag, classNavCard).
			AppendInt32(1, 1001).
			AppendInt64(2, 1).
			AppendMessage(101, banner).
			Bytes()

		app := protostuff.NewEncoder().
			AppendInt64(1, 100).
			AppendString(3, "DemoApp").
			Bytes()
		listCard := protostuff.NewEncoder().
			AppendString(protostuff.PolyClassTag, classResourceListCard).
			AppendInt32(1, 1002).
			AppendInt64(2, 2).
			AppendString(101, "推荐应用").
			AppendMessage(103, app).
			Bytes()

		view := protostuff.NewEncoder().
			AppendString(1, "Home").
			AppendInt32(2, 100).
			AppendMessage(3, navCard).
			AppendMessage(3, listCard).
			AppendInt32(4, 1).
			Bytes()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(view)
	})

	client := NewStoreClient("https://mock.store.local", 5*time.Second, httpClient)
	got, err := client.GetHome(context.Background(), domain.DefaultDeviceProfile(), domain.HomeParams{Start: 0, Size: 10})
	if err != nil {
		t.Fatalf("GetHome failed: %v", err)
	}
	if got.PageKey != 100 || got.IsEnd != 1 || len(got.Cards) != 2 {
		t.Fatalf("unexpected view layer: %+v", got)
	}
	if got.Cards[0].Type != classNavCard || len(got.Cards[0].Banners) != 1 || got.Cards[0].Banners[0].Title != "Top Banner" {
		t.Fatalf("unexpected NavCard: %+v", got.Cards[0])
	}
	if got.Cards[1].Type != classResourceListCard || got.Cards[1].Title != "推荐应用" || len(got.Cards[1].Resources) != 1 {
		t.Fatalf("unexpected ResourceListCard: %+v", got.Cards[1])
	}
}

func TestStoreClientNotFoundOn204(t *testing.T) {
	httpClient := newInMemoryHTTPClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	client := NewStoreClient("https://mock.store.local", 5*time.Second, httpClient)
	_, err := client.GetAppDetailByID(context.Background(), domain.DefaultDeviceProfile(), domain.AppDetailByIDParams{
		AppID:   999999999,
		AppType: 1,
		Module:  "search",
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStoreClientCheckUpdatesAndOrders(t *testing.T) {
	httpClient := newInMemoryHTTPClient(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Errorf("expected protostuff request body on %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/x-protostuff; charset=UTF-8" {
			t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}

		switch r.URL.Path {
		case "/watch/update-info":
			upg := protostuff.NewEncoder().
				AppendInt64(1, 31617863).
				AppendString(4, "com.tencent.watchwechat").
				AppendString(101, "Bug fixes").
				AppendInt32(102, 1).
				Bytes()
			resp := protostuff.NewEncoder().AppendMessage(1, upg).Bytes()
			_, _ = w.Write(resp)
		case "/watch/createOrder":
			vo := protostuff.NewEncoder().
				AppendString(1, "ord-123").
				AppendString(2, "https://pay/qr").
				Bytes()
			resp := protostuff.NewEncoder().
				AppendString(1, "200").
				AppendString(2, "ok").
				AppendMessage(3, vo).
				Bytes()
			_, _ = w.Write(resp)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	client := NewStoreClient("https://mock.store.local", 5*time.Second, httpClient)
	upd, err := client.CheckUpdates(context.Background(), domain.DefaultDeviceProfile(), []domain.InstalledApp{
		{PkgName: "com.tencent.watchwechat", VerCode: "1"},
	})
	if err != nil {
		t.Fatalf("CheckUpdates failed: %v", err)
	}
	if len(upd.Upgrades) != 1 || upd.Upgrades[0].UpgradeFlag != 1 || upd.Upgrades[0].UpdateDesc != "Bug fixes" {
		t.Fatalf("unexpected upgrades: %+v", upd)
	}

	ord, err := client.CreateOrder(context.Background(), domain.DefaultDeviceProfile(), domain.CreateOrderInput{
		AppID:       37627586,
		Price:       600,
		ProductName: "Lab",
		ProductDesc: "com.lab",
		Token:       "tok",
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}
	if ord.Code != "200" || ord.Data == nil || ord.Data.OrderID != "ord-123" {
		t.Fatalf("unexpected order response: %+v", ord)
	}
}
