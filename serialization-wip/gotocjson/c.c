#include "convert_go_to_c.h"
#include "config.h"

/*
typedef struct {
    string Entrypoint;
} config_GroundworkAction;
*/
json_t *config_GroundworkAction_as_JSON(const config_GroundworkAction *config_GroundworkAction) {
    printf(FILE_LINE "at start of config_GroundworkAction_as_JSON, config_GroundworkAction is %p\n", config_GroundworkAction);
    printf(FILE_LINE "config_GroundworkAction->Entrypoint is %p\n", config_GroundworkAction->Entrypoint);
    json_t *json;
    json = json_pack("{s:s?}"
        , "Entrypoint", config_GroundworkAction->Entrypoint
    );
    printf(FILE_LINE "at end   of config_GroundworkAction_as_JSON, json is %p\n", json);
    return json;
}

/*
typedef struct {
    string ControllerAddr;
    string ControllerCertFile;
    string ControllerKeyFile;
    string NATSFilestoreDir;
    string NATSStoreType;
    bool StartController;
    bool StartNATS;
    bool StartTransport;
} config_AgentConfig;
*/
json_t *config_AgentConfig_as_JSON(const config_AgentConfig *config_AgentConfig) {
    printf(FILE_LINE "at start of config_AgentConfig_as_JSON\n");
    json_t *json;
    json = json_pack("{s:s? s:s? s:s? s:s? s:s? s:b s:b s:b}"
	, "ControllerAddr",     config_AgentConfig->ControllerAddr
	, "ControllerCertFile", config_AgentConfig->ControllerCertFile
	, "ControllerKeyFile",  config_AgentConfig->ControllerKeyFile
	, "NATSFilestoreDir",   config_AgentConfig->NATSFilestoreDir
	, "NATSStoreType",      config_AgentConfig->NATSStoreType
	, "StartController",    config_AgentConfig->StartController
	, "StartNATS",          config_AgentConfig->StartNATS
	, "StartTransport",     config_AgentConfig->StartTransport
    );
    printf(FILE_LINE "at end   of config_AgentConfig_as_JSON\n");
    return json;
}

/*
typedef struct {
    string Host;
    string Account;
    string Password;
    string Token;
    string AppName;
} config_GroundworkConfig;
*/
json_t *config_GroundworkConfig_as_JSON(const config_GroundworkConfig *config_GroundworkConfig) {
    printf(FILE_LINE "at start of config_GroundworkConfig_as_JSON\n");
    json_t *json;
    json = json_pack("{s:s? s:s? s:s? s:s? s:s?}"
        , "Host",     config_GroundworkConfig->Host
        , "Account",  config_GroundworkConfig->Account
        , "Password", config_GroundworkConfig->Password
        , "Token",    config_GroundworkConfig->Token
        , "AppName",  config_GroundworkConfig->AppName
    );
    printf(FILE_LINE "at end   of config_GroundworkConfig_as_JSON\n");
    return json;
}

/*
typedef struct {
    config_GroundworkAction Connect;
    config_GroundworkAction Disconnect;
    config_GroundworkAction SynchronizeInventory;
    config_GroundworkAction SendResourceWithMetrics;
    config_GroundworkAction ValidateToken;
} config_GroundworkActions;
*/
json_t *config_GroundworkActions_as_JSON(const config_GroundworkActions *config_GroundworkActions) {
    printf(FILE_LINE "at start of config_GroundworkActions_as_JSON\n");
    printf(FILE_LINE "                Connect.Entrypoint = %p\n", config_GroundworkActions->Connect.Entrypoint);
    printf(FILE_LINE "             Disconnect.Disconnect = %p\n", config_GroundworkActions->Disconnect.Entrypoint);
    printf(FILE_LINE "   SynchronizeInventory.Entrypoint = %p\n", config_GroundworkActions->SynchronizeInventory.Entrypoint);
    printf(FILE_LINE "SendResourceWithMetrics.Entrypoint = %p\n", config_GroundworkActions->SendResourceWithMetrics.Entrypoint);
    printf(FILE_LINE "          ValidateToken.Entrypoint = %p\n", config_GroundworkActions->ValidateToken.Entrypoint);
    json_t *json;
    json = json_pack("{s:o? s:o? s:o? s:o? s:o?}"
        , "Connect",                 config_GroundworkAction_as_JSON( &config_GroundworkActions->Connect                 )
        , "Disconnect",              config_GroundworkAction_as_JSON( &config_GroundworkActions->Disconnect              )
        , "SynchronizeInventory",    config_GroundworkAction_as_JSON( &config_GroundworkActions->SynchronizeInventory    )
        , "SendResourceWithMetrics", config_GroundworkAction_as_JSON( &config_GroundworkActions->SendResourceWithMetrics )
        , "ValidateToken",           config_GroundworkAction_as_JSON( &config_GroundworkActions->ValidateToken           )
    );
    printf(FILE_LINE "at end   of config_GroundworkActions_as_JSON\n");
    return json;
}

/*
typedef struct {
    config_AgentConfig AgentConfig;
    config_GroundworkConfig GroundworkConfig;
    config_GroundworkActions GroundworkActions;
} config_Config;
*/
json_t *config_Config_as_JSON(const config_Config *config_Config) {
    json_t *json;
    if (config_Config == NULL) {
        printf(FILE_LINE "config_Config is NULL\n");
    }
    else {
        printf(FILE_LINE "config_Config is not NULL\n");
	/*
	if (config_Config->AgentConfig == NULL) {
	    printf(FILE_LINE "config_Config->AgentConfig is NULL\n");
	}
	if (config_Config->GroundworkConfig == NULL) {
	    printf(FILE_LINE "config_Config->GroundworkConfig is NULL\n");
	}
	if (config_Config->GroundworkActions == NULL) {
	    printf(FILE_LINE "config_Config->GroundworkActions is NULL\n");
	}
	*/
	printf(FILE_LINE "before json_pack() in config_Config_as_JSON\n");
	json = json_pack("{s:o? s:o? s:o?}"
	    , "AgentConfig",             config_AgentConfig_as_JSON( &config_Config->AgentConfig )
	    , "GroundworkConfig",   config_GroundworkConfig_as_JSON( &config_Config->GroundworkConfig )
	    , "GroundworkActions", config_GroundworkActions_as_JSON( &config_Config->GroundworkActions )
	);
    }
    printf(FILE_LINE " after json_pack() in config_Config_as_JSON\n");
    return json;
}
