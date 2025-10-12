package kernel

type UserID string

func NewUserID(id string) UserID { return UserID(id) }
func (u UserID) String() string  { return string(u) }
func (u UserID) IsEmpty() bool   { return string(u) == "" }

type TenantID string

func NewTenantID(id string) TenantID { return TenantID(id) }
func (t TenantID) String() string    { return string(t) }
func (t TenantID) IsEmpty() bool     { return string(t) == "" }

type RoleID string

func NewRoleID(id string) RoleID { return RoleID(id) }
func (r RoleID) String() string  { return string(r) }
func (r RoleID) IsEmpty() bool   { return string(r) == "" }

type MessageID string

func NewMessageID(id string) MessageID { return MessageID(id) }
func (r MessageID) String() string     { return string(r) }
func (r MessageID) IsEmpty() bool      { return string(r) == "" }

type ChannelID string

func NewChannelID(id string) ChannelID { return ChannelID(id) }
func (r ChannelID) String() string     { return string(r) }
func (r ChannelID) IsEmpty() bool      { return string(r) == "" }

type WorkflowID string

func NewWorkflowID(id string) WorkflowID { return WorkflowID(id) }
func (r WorkflowID) String() string      { return string(r) }
func (r WorkflowID) IsEmpty() bool       { return string(r) == "" }

type ParserID string

func NewParserID(id string) ParserID { return ParserID(id) }
func (r ParserID) String() string    { return string(r) }
func (r ParserID) IsEmpty() bool     { return string(r) == "" }

type ToolID string

func NewToolID(id string) ToolID { return ToolID(id) }
func (r ToolID) String() string  { return string(r) }
func (r ToolID) IsEmpty() bool   { return string(r) == "" }

type SessionID string

func NewSessionID(id string) SessionID { return SessionID(id) }
func (r SessionID) String() string     { return string(r) }
func (r SessionID) IsEmpty() bool      { return string(r) == "" }
