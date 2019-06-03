module github.com/lakstap/go-atk

go 1.12

replace github.com/testcontainers/testcontainer-go v0.0.2 => github.com/testcontainers/testcontainers-go v0.0.4

replace github.com/golang/lint v0.0.0-20190409202823-959b441ac422 => github.com/golang/lint v0.0.0-20190409202823-5614ed5bae6fb75893070bdc0996a68765fdd275

replace github.com/golang/lint v0.0.0-20190313153728-d0100b6bd8b3 => github.com/golang/lint v0.0.0-20190409202823-5614ed5bae6fb75893070bdc0996a68765fdd275

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d

require (
	github.com/coreos/go-oidc v2.0.0+incompatible
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/go-log/log v0.1.0
	github.com/gogo/protobuf v1.2.1
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/uuid v1.1.1
	github.com/grpc-ecosystem/grpc-gateway v1.9.0
	github.com/micro/cli v0.1.0
	github.com/micro/go-config v1.1.0
	github.com/micro/go-grpc v1.0.1
	github.com/micro/go-log v0.1.0
	github.com/micro/go-micro v1.1.0
	github.com/micro/go-plugins v1.1.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/rakyll/statik v0.1.6
	github.com/rs/cors v1.6.0
	golang.org/x/oauth2 v0.0.0-20190523182746-aaccbc9213b0
	google.golang.org/grpc v1.21.0
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce
	gopkg.in/resty.v1 v1.12.0
	k8s.io/api v0.0.0-20190405172450-8fc60343b75c
	k8s.io/apimachinery v0.0.0-20190405172352-ba051b3c4d9d
	k8s.io/client-go v11.0.0+incompatible
)
