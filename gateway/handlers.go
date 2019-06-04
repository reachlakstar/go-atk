package gateway

import (
	"net/http"
	"github.com/golang/glog"
	"strings"
	"path"
	"mime"
	"context"
	_ "github.com/lakstap/go-atk/swagger"
	"github.com/rakyll/statik/fs"
	"crypto/tls"
	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"fmt"
	"github.com/rs/cors"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc"
	"time"
	"github.com/google/uuid"
	"crypto/rsa"
)

type UserInfo struct {
	Email         string   `json:"email"`
	MsId          string   `json:"preferred_username"`
	EmailVerified bool     `json:"email_verified"`
	Groups        []string `json:"groups"`
}
type userCtxKey struct{}
type isAdminCtxKey struct{}

const (
	userKeyStr    = "User"
	isAdminKeyStr = "IsAdmin"
)

var (
	clientID          = ""
	keyCloakIssuerUrl = ""
	verifyKey         *rsa.PublicKey
	environment       = "dev"
)

// swaggerServer returns swagger specification files located under "/swagger/"
func ServeSwaggerJSON(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		glog.Infof("Serving %s", r.URL.Path)
		p := strings.TrimPrefix(r.URL.Path, "/swagger/")
		p = path.Join(dir, p)
		fmt.Println("SwaggerServerJSON:: Servicing Swagger Json file  ...")
		http.ServeFile(w, r, p)
	}
}

func SwaggerServer(mux *http.ServeMux) {
	mime.AddExtensionType(".svg", "image/svg+xml")

	swaggerFS, err := fs.New()
	if err != nil {
		glog.Error("Failed to load the directory")
	}

	// Expose files in third_party/swagger-ui/ on <host>/swagger-ui
	prefix := "/swagger/"
	mux.Handle(prefix, http.StripPrefix(prefix, http.FileServer(swaggerFS)))
}

func WithClientUnaryInterceptor(env string) grpc.DialOption {
	environment = env
	return grpc.WithUnaryInterceptor(clientInterceptor)
}

func clientInterceptor(
	ctx context.Context,
	method string,
	req interface{},
	reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	// Logic before invoking the invoker
	start := time.Now()
	_, err := uuid.NewUUID()

	// Calls the invoker to execute RPC
	err = invoker(ctx, method, req, reply, cc, opts...)
	// Logic after invoking the invoker
	glog.Infof("Invoked RPC method=%s; Duration=%s; Error=%v;", method,
		time.Since(start), err)

	return err
}

func DefaultAuthMiddleware(ctxt context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), userCtxKey{}, "admin"))
		next.ServeHTTP(w, r)
	})
}
func AuthMiddleware(ctxt context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			sslcli := &http.Client{Transport: tr}
			ctx := context.WithValue(ctxt, oauth2.HTTPClient, sslcli)

			authorizationHeader := r.Header.Get("authorization")
			if authorizationHeader != "" {
				bearerToken := strings.Split(authorizationHeader, " ")

				if len(bearerToken) == 2 {
					/* verify the token from keycloak issuer url */
					userInfo, err := verifyBearerToken(ctx, keyCloakIssuerUrl, bearerToken, r)
					if err != nil {

						http.Error(w, "idTokenVerifier: Failed to verify ID Token: "+err.Error(), http.StatusUnauthorized)
						return

					} else {
						r = r.WithContext(context.WithValue(r.Context(), userCtxKey{}, userInfo.MsId))
						//r = r.WithContext(context.WithValue(r.Context(), isAdminCtxKey{}, userInfo.Groups))
					}

				}
			} else {
				http.Error(w, "An authorization header is required: ", http.StatusUnauthorized)
				return
			}
		next.ServeHTTP(w, r)
	})
}

/*
 * verifyBearerToken
 */
func verifyBearerToken(ctx context.Context, issuerUrl string, bearerToken []string, r *http.Request) (*UserInfo, error) {
	idTokenVerifier, err := initVerifier(ctx, &oidc.Config{ClientID: clientID}, issuerUrl)

	if err != nil {
		return nil, err
	}
	idToken, err := idTokenVerifier.Verify(ctx, bearerToken[1])
	if err != nil {
		return nil, err
	}
	userInfo := &UserInfo{}
	if err := idToken.Claims(userInfo); err != nil {
		return nil, err
	}
	return userInfo, nil
}

func initVerifier(ctx context.Context, config *oidc.Config, iss string) (*oidc.IDTokenVerifier, error) {
	provider, err := oidc.NewProvider(ctx, iss)
	if err != nil {
		return nil, fmt.Errorf("init verifier failed: %v", err)
	}
	return provider.Verifier(config), nil
}

func SetupGlobalMiddleware(handler http.Handler) http.Handler {
	handleCORS := cors.Default().Handler

	return handleCORS(handler)
}

func ForwardAuthenticationMetadata(ctx context.Context, r *http.Request) metadata.MD {
	md := metadata.MD{}
	if user := r.Context().Value(userCtxKey{}); user != nil {
		md.Set(userKeyStr, user.(string))
		//groups := r.Context().Value(isAdminCtxKey{}).([]string)

		md.Set(isAdminKeyStr, "false")
		/*for _, grp := range groups {
			if strings.EqualFold(grp, ADMIN_group_NAME) {
				md.Set(isAdminKeyStr, "true")
			}
		}*/

	}
	return md
}
