package atk

import (
	"github.com/micro/go-log"

	"github.com/micro/go-micro"
	"github.com/micro/go-grpc"
	"github.com/micro/cli"
	"github.com/lakstap/go-atk/database/config"
	"strings"
	"time"
	"github.com/patrickmn/go-cache"

	"github/lakstap/go-atk/database"
)

// Options is a set of options to be passed to Run
type ATKGrpcServiceOption struct {
	// Grpc Service Name
	ServiceName string

	// Version name for the service
	Version string

	// Type of GRPC Service ( Database )
	ServiceType string
}

// ATK Grpc Service
type ATKGrpcService struct {
	Options ATKGrpcServiceOption

	Service micro.Service

	Session *mdb.DatabaseSession

	ATKCache *cache.Cache
}

// New ATK GRPC Service returns a new grpc with default values.
func NewATKGrpcService(opts ATKGrpcServiceOption) *ATKGrpcService {

	atkService := &ATKGrpcService{
		Options:  opts,
		ATKCache: cache.New(5*time.Minute, 10*time.Minute),
		Service: grpc.NewService(
			micro.Flags(
				cli.StringFlag{
					Name:  "db_config_path",
					Usage: "JSON  file path for database configuration",
					Value: "config/config.json",
				},
			),
			micro.Name(opts.ServiceName), //"go.micro.srv.atk.Grpc.project"
			micro.Version(opts.Version),
			//gsrv.Options(gogrpc.UnaryInterceptor(unaryInterceptor)),

		),
	}

	// Initialize service based on the type
	if strings.EqualFold(opts.ServiceType, "database") {
		atkService.Service.Init(
			micro.Action(func(c *cli.Context) {
				log.Log("Reading the config data from the configuration file..")
				dbConfigPath := c.String("db_config_path")
				if len(dbConfigPath) > 0 {
					log.Log("Parsing the Database config file...", dbConfigPath)
					// Read the Config
					dbConfig, err := config.ReadConfig(dbConfigPath)
					if err != nil {
						log.Fatal(err)
					}
					log.Logf("Read the configuration file (%s)..", dbConfig.Address)
					// create the session
					atkService.Session, err = mdb.GetDBSession(dbConfig)
					if err != nil {
						log.Fatal(err)
					}
				}
			}),
		)
	}

	return atkService
}

func (e *ATKGrpcService) RunATKGrpcService() error {

	// Run service
	if err := e.Service.Run(); err != nil {
		log.Fatal(err)
	}
	return nil
}
