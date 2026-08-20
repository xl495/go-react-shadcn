package httpserver

// Application error codes. HTTP status is separate; these go in JSON "code".
const (
	CodeOK = 0

	CodeInvalidBody        = 40001
	CodeCaptchaRequired    = 40002
	CodeInvalidCaptcha     = 40003
	CodeUserPassRequired   = 40010
	CodeInvalidRoleIDs     = 40011
	CodeInvalidUserBody    = 40012
	CodeCannotDeleteSeed   = 40013
	CodeInvalidAssignBody  = 40014
	CodeInvalidDictValue   = 40015
	CodePasswordTooShort   = 40016
	CodeRoleNameRequired   = 40020
	CodeInvalidPermIDs     = 40021
	CodeInvalidRoleBody    = 40022
	CodeCannotDeleteRole   = 40023
	CodeInvalidRolePerms   = 40024
	CodePermRequired       = 40030
	CodeInvalidPermBody    = 40031
	CodeProfileBody        = 40040
	CodeWrongPassword      = 40041
	CodePasswordRequired   = 40042
	CodeNewPasswordShort   = 40043
	CodeResetTokenInvalid  = 40044
	CodeEmailRequired      = 40045
	CodeMailIncomplete     = 40080
	CodeMailRecipient      = 40081
	CodeInvalidTimezone    = 40082
	CodeInvalidUnsubToken  = 40083
	CodeMailJobCannotRetry = 40084
	CodeMailCampaignState  = 40085
	CodeAvatarRequired     = 40050
	CodeAvatarTooLarge     = 40051
	CodeAvatarType         = 40052
	CodeDictRequired       = 40060
	CodeInvalidDictBody    = 40061
	CodeDictItemRequired   = 40062
	CodeInvalidItemBody    = 40063
	CodeConfigRequired     = 40070
	CodeInvalidConfigBody  = 40071
	CodeDeptRequired       = 40090
	CodeInvalidDeptBody    = 40091
	CodeDeptHasChildren    = 40092

	CodeMissingToken   = 40101
	CodeInvalidToken   = 40102
	CodeBadCredentials = 40103

	CodeForbidden     = 40301
	CodeAccountLocked = 40310

	CodeUserMissingMe        = 40401
	CodeUserNotFound         = 40410
	CodeRoleNotFound         = 40420
	CodePermNotFound         = 40430
	CodeDictNotFound         = 40460
	CodeDictItemMissing      = 40461
	CodeConfigNotFound       = 40470
	CodeMailJobNotFound      = 40480
	CodeMailCampaignNotFound = 40481
	CodeDeptNotFound         = 40490

	CodeUserExists   = 40910
	CodeRoleExists   = 40920
	CodePermExists   = 40930
	CodeDictExists   = 40960
	CodeDictItemDup  = 40961
	CodeConfigExists = 40970
	CodeDeptExists   = 40990

	CodeLoginRateLimited  = 42901
	CodeForgotRateLimited = 42902

	CodeCasbinCheck     = 50001
	CodeCaptchaIssue    = 50002
	CodeTokenIssue      = 50003
	CodeListUsers       = 50010
	CodeHashPassword    = 50011
	CodeAssignRoles     = 50012
	CodeSyncRBAC        = 50013
	CodeUpdateUser      = 50014
	CodeDeleteUser      = 50016
	CodeListRoles       = 50020
	CodeAssignPerms     = 50021
	CodeUpdateRole      = 50023
	CodeDeleteRole      = 50025
	CodeListPerms       = 50030
	CodeUpdatePerm      = 50031
	CodeDetachPerm      = 50032
	CodeDeletePerm      = 50033
	CodeStats           = 50040
	CodeUpdateProfile   = 50041
	CodeChangePassword  = 50042
	CodeStoreAvatar     = 50050
	CodeListDicts       = 50060
	CodeUpdateDict      = 50061
	CodeDeleteDictItems = 50062
	CodeDeleteDict      = 50063
	CodeListDictItems   = 50064
	CodeUpdateDictItem  = 50065
	CodeDeleteDictItem  = 50066
	CodeListConfigs     = 50070
	CodeUpdateConfig    = 50071
	CodeDeleteConfig    = 50072
	CodeClearLogs       = 50081
	CodePurgeLogs       = 50084
	CodeListDepts       = 50090
	CodeUpdateDept      = 50091
	CodeDeleteDept      = 50092
	CodeListMenus       = 50093

	CodeSendMail     = 50085
	CodeUnhealthy    = 50301
	CodeMailDisabled = 50310
)
