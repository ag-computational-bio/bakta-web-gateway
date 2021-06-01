package gateways

import (
	"log"

	"context"
	"fmt"
	"net/http"
	"strings"

	api "github.com/ag-computational-bio/bakta-web-api-go/bakta/web/api/proto/v1"
	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/spf13/viper"

	"google.golang.org/grpc"
)

// grpcHandlerFunc returns an http.Handler that delegates to grpcServer on incoming gRPC
// connections or otherHandler otherwise. Copied from cockroachdb.
func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is a partial recreation of gRPC's internal checks https://github.com/grpc/grpc-go/pull/514/files#diff-95e9a25b738459a2d3030e1e6fa2a718R61
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	})
}

// StartETLGateway Starts the gateway server for the ETL component
func StartGateway() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	gwmux := runtime.NewServeMux()

	grpcEndpointHost := viper.GetString("Config.Gateway.GRPCEndpointHost")
	grpcEndpointPort := viper.GetInt("Config.Gateway.GRPCEndpointPort")

	opts := []grpc.DialOption{grpc.WithInsecure()}

	r := gin.Default()

	r.Any("/api/*any", gin.WrapF(gwmux.ServeHTTP))

	swagger_fs := http.FS(api.GetSwaggerEmbedded())
	r.StaticFS("/swaggerjson", swagger_fs)
	fs := http.FileSystem(http.Dir("www/swagger-ui/"))

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/swagger-ui/")
	})

	r.StaticFS("/swagger-ui/", fs)

	err := api.RegisterBaktaJobsHandlerFromEndpoint(ctx, gwmux, fmt.Sprintf("%v:%v", grpcEndpointHost, grpcEndpointPort), opts)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	port := viper.GetInt("Config.Gateway.Port")

	return r.Run(fmt.Sprintf(":%v", port))
}
