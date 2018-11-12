package app

import (
	"flag"

	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/flowcontrol"
)

type RestClientOptions struct {
	APIQPS               float64
	ClientConfigFileFrom string
	ClientConfigFileName string
	ClientContext        string
}

func BindRestClientFlags(o *RestClientOptions, fs *flag.FlagSet) {
	fs.Float64Var(&o.APIQPS, "api-qps", 5, "Maximum queries per second when talking to Kubernetes API")
	fs.StringVar(&o.ClientConfigFileFrom, "client-config-from", "in-cluster",
		"Source of REST client configuration. 'in-cluster' (default) and 'file' are valid options.")
	fs.StringVar(&o.ClientConfigFileName, "client-config-file-name", "",
		"Load REST client configuration from the specified Kubernetes config file. This is only applicable if --client-config-from=file is set.")
	fs.StringVar(&o.ClientContext, "client-config-context", "",
		"Context to use for REST client configuration. This is only applicable if --client-config-from=file is set.")
}

func LoadRestClientConfig(userAgent string, options RestClientOptions) (*rest.Config, error) {
	var config *rest.Config
	var err error

	switch options.ClientConfigFileFrom {
	case "in-cluster":
		config, err = rest.InClusterConfig()
	case "file":
		var configAPI *clientcmdapi.Config
		configAPI, err = clientcmd.LoadFromFile(options.ClientConfigFileName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load REST client configuration from file %q", options.ClientConfigFileName)
		}
		config, err = clientcmd.NewDefaultClientConfig(*configAPI, &clientcmd.ConfigOverrides{
			CurrentContext: options.ClientContext,
		}).ClientConfig()
	default:
		err = errors.New("invalid value for 'client config from' parameter")
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load REST client configuration from %q", options.ClientConfigFileFrom)
	}
	config.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(float32(options.APIQPS), int(options.APIQPS*1.5))
	config.UserAgent = userAgent
	return config, nil
}
