package translator

import (
	errors "github.com/rotisserie/eris"
	extauthv1 "github.com/solo-io/gloo/projects/gloo/pkg/api/v1/enterprise/options/extauth/v1"
	"github.com/solo-io/gloo/projects/gloo/pkg/plugins"
	"github.com/solo-io/solo-kit/pkg/api/v2/reporter"
)

func (t *translatorInstance) verifyAuthConfigs(params plugins.Params, reports reporter.ResourceReports) {
	authConfigs := params.Snapshot.AuthConfigs
	for _, ac := range authConfigs {
		configs := ac.GetConfigs()
		if len(configs) == 0 {
			reports.AddError(ac, errors.Errorf("No configurations for auth config %v", ac.Metadata.String()))
		}
		for _, conf := range configs {
			switch cfg := conf.AuthConfig.(type) {
			case *extauthv1.AuthConfig_Config_BasicAuth:
				if cfg.BasicAuth.GetApr() == nil || cfg.BasicAuth.GetRealm() == "" {
					reports.AddError(ac, errors.Errorf("Invalid configurations for basic auth config %v", ac.Metadata.String()))
				}
			case *extauthv1.AuthConfig_Config_Oauth:
				if cfg.Oauth.GetAppUrl() == "" {
					reports.AddError(ac, errors.Errorf("Invalid configurations for basic auth config %v", ac.Metadata.String()))
				}
			case *extauthv1.AuthConfig_Config_ApiKeyAuth:
				if len(cfg.ApiKeyAuth.GetLabelSelector())+len(cfg.ApiKeyAuth.GetLabelSelector()) == 0 {
					reports.AddError(ac, errors.Errorf("Invalid configurations for apikey auth config %v", ac.Metadata.String()))
				}
			case *extauthv1.AuthConfig_Config_PluginAuth:
				if cfg.PluginAuth.GetConfig() == nil {
					reports.AddError(ac, errors.Errorf("Invalid configurations for plugin auth config %v", ac.Metadata.String()))
				}
			case *extauthv1.AuthConfig_Config_OpaAuth: // no way to verify this from just the config
			case *extauthv1.AuthConfig_Config_Ldap:
				if cfg.Ldap.GetAddress() == "" {
					reports.AddError(ac, errors.Errorf("Invalid configurations for ldap auth config %v", ac.Metadata.String()))
				}
			default:
				reports.AddError(ac, errors.Errorf("Unknown Auth Config type for %v", ac.Metadata.String()))
			}
		}
	}
}
