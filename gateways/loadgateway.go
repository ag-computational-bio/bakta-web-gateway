package gateways

import (
	"log"

	"github.com/ag-computational-bio/bakta-web-api/go/api"
	"github.com/ag-computational-bio/bakta-web-api/go/swaggerhandler"
	"github.com/gin-contrib/cors"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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
func StartETLGateway() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	gwmux := runtime.NewServeMux()

	grpcEndpointHost := viper.GetString("Config.Gateway.GRPCEndpointHost")
	grpcEndpointPort := viper.GetInt("Config.Gateway.GRPCEndpointPort")

	opts := []grpc.DialOption{grpc.WithInsecure()}

	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"https://ui.bakta.ingress.rancher2.computational.bio/", "https://restapi.bakta.ingress.rancher2.computational.bio"}
	config.AllowCredentials = true
	config.AddAllowHeaders("authorization")

	r.Use(cors.New(config))

	r.Any("/api/*any", gin.WrapF(gwmux.ServeHTTP))

	r.GET("/swaggerjson", func(c *gin.Context) {
		c.Data(200, "application/json", swaggerhandler.SwaggerJSON)
	})

	err := api.RegisterBaktaJobsHandlerFromEndpoint(ctx, gwmux, fmt.Sprintf("%v:%v", grpcEndpointHost, grpcEndpointPort), opts)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	swaggerDir := viper.GetString("Config.Swagger.Path")

	fs := http.FileSystem(http.Dir(swaggerDir))
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/swagger-ui/")
	})
	r.StaticFS("/swagger-ui/", fs)

	port := viper.GetInt("Config.Gateway.Port")

	return r.Run(fmt.Sprintf(":%v", port))
}
