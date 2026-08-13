module loadmon

go 1.25.0

require (
	github.com/HewlettPackard/oneview-golang v11.3.0+incompatible
	github.com/beego/beego/v2 v2.3.10
	github.com/gorilla/sessions v1.4.0
	github.com/jcmturner/goidentity/v6 v6.0.1
	github.com/jcmturner/gokrb5/v8 v8.4.4
	github.com/lib/pq v1.10.9
	github.com/lovi-cloud/go-dorado-sdk v0.9.0
	github.com/mattn/go-sqlite3 v1.14.28
	github.com/nixys/nxs-go-zabbix/v5 v5.2.0
	github.com/orcaman/concurrent-map/v2 v2.0.1
	github.com/prometheus/client_golang v1.24.1
	github.com/spa-nsk/oneview2 v0.0.0-00010101000000-000000000000
	github.com/twinj/uuid v1.0.0
	github.com/vmware/govmomi v0.55.1
	github.com/wcharczuk/go-chart v2.0.1+incompatible
	github.com/xuri/excelize/v2 v2.11.0
	golang.org/x/crypto v0.54.0
	golang.org/x/net v0.57.0
	google.golang.org/grpc v1.83.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blend/go-sdk v1.20240719.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/docker/machine v0.16.2 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/myesui/uuid v1.0.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.70.1 // indirect
	github.com/prometheus/procfs v0.21.1 // indirect
	github.com/richardlehane/mscfb v1.0.7 // indirect
	github.com/richardlehane/msoleps v1.0.6 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/shiena/ansicolor v0.0.0-20230509054315-a9deabde6e02 // indirect
	github.com/tiendc/go-deepcopy v1.7.2 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/xuri/efp v0.0.1 // indirect
	github.com/xuri/nfp v0.0.2-0.20250530014748-2ddeb826f9a9 // indirect
	golang.org/x/image v0.38.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	gopkg.in/stretchr/testify.v1 v1.2.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/beego/beego/v2 => ./beego/v2

replace github.com/spa-nsk/oneview2 => ./oneview2
