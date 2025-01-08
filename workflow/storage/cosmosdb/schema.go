package cosmosdb

var tables = []string{
	planSchema,
	blocksSchema,
	checksSchema,
	sequencesSchema,
	actionsSchema,
}

type plansEntry struct {
	PartitionKey   string `json:"partitionKey,omitempty"`
	ID             string `json:"id,omitempty"`
	GroupID        string `json:"groupID,omitempty"`
	Name           string `json:"name,omitempty"`
	Descr          string `json:"descr,omitempty"`
	Meta           []byte `json:"meta,omitempty"`
	BypassChecks   string `json:"bypassChecks,omitempty"`
	PreChecks      string `json:"preChecks,omitempty"`
	PostChecks     string `json:"postChecks,omitempty"`
	ContChecks     string `json:"contChecks,omitempty"`
	DeferredChecks string `json:"deferredChecks,omitempty"`
	Blocks         string `json:"blocks,omitempty"`
	StateStatus    int64  `json:"stateStatus,omitempty"`
	StateStart     int64  `json:"stateStart,omitempty"`
	StateEnd       int64  `json:"stateEnd,omitempty"`
	SubmitTime     int64  `json:"submitTime,omitempty"`
	Reason         int64  `json:"reason,omitempty"`
}

var planSchema = `
CREATE Table If Not Exists plans (
	id TEXT PRIMARY KEY,
	group_id TEXT NOT NULL,
	name TEXT NOT NULL,
	descr TEXT NOT NULL,
	meta BLOB,
	bypasschecks TEXT,
	prechecks TEXT,
	postchecks TEXT,
	contchecks TEXT,
	deferredchecks TEXT,
	blocks BLOB NOT NULL,
	state_status INTEGER NOT NULL,
	state_start INTEGER NOT NULL,
	state_end INTEGER NOT NULL,
	submit_time INTEGER NOT NULL,
	reason INTEGER
);`

type blocksEntry struct {
	PartitionKey      string `json:"partitionKey,omitempty"`
	ID                string `json:"id,omitempty"`
	PlanID            string `json:"planID,omitempty"`
	Key               string `json:"key,omitempty"`
	Name              string `json:"name,omitempty"`
	Descr             string `json:"descr,omitempty"`
	Pos               int64  `json:"pos,omitempty"`
	EntranceDelay     int64  `json:"entranceDelay,omitempty"`
	ExitDelay         int64  `json:"exitDelay,omitempty"`
	BypassChecks      string `json:"bypassChecks,omitempty"`
	PreChecks         string `json:"preChecks,omitempty"`
	PostChecks        string `json:"postChecks,omitempty"`
	ContChecks        string `json:"contChecks,omitempty"`
	DeferredChecks    string `json:"deferredChecks,omitempty"`
	Sequences         string `json:"sequences,omitempty"`
	Concurrency       int64  `json:"concurrency,omitempty"`
	ToleratedFailures int64  `json:"toleratedFailures,omitempty"`
	StateStatus       int64  `json:"stateStatus,omitempty"`
	StateStart        int64  `json:"stateStart,omitempty"`
	StateEnd          int64  `json:"stateEnd,omitempty"`
}

var blocksSchema = `
CREATE Table If Not Exists blocks (
    id TEXT PRIMARY KEY,
    key TEXT,
    plan_id BLOB NOT NULL,
    name TEXT NOT NULL,
    descr TEXT NOT NULL,
    pos INTEGER NOT NULL,
    entrancedelay INTEGER NOT NULL,
    exitdelay INTEGER NOT NULL,
    bypasschecks TEXT,
    prechecks TEXT,
    postchecks TEXT,
    contchecks TEXT,
    deferredchecks TEXT,
    sequences BLOB NOT NULL,
    concurrency INTEGER NOT NULL,
    toleratedfailures INTEGER NOT NULL,
    state_status INTEGER NOT NULL,
    state_start INTEGER NOT NULL,
    state_end INTEGER NOT NULL
);`

type checksEntry struct {
	PartitionKey string `json:"partitionKey,omitempty"`
	ID           string `json:"id,omitempty"`
	Key          string `json:"key,omitempty"`
	PlanID       string `json:"planID,omitempty"`
	Actions      string `json:"actions,omitempty"`
	Delay        int64  `json:"delay,omitempty"`
	StateStatus  int64  `json:"stateStatus,omitempty"`
	StateStart   int64  `json:"stateStart,omitempty"`
	StateEnd     int64  `json:"stateEnd,omitempty"`
}

var checksSchema = `
CREATE Table If Not Exists checks (
    id TEXT PRIMARY KEY,
    key TEXT,
    plan_id TEXT NOT NULL,
    actions BLOB NOT NULL,
    delay INTEGER NOT NULL,
    state_status INTEGER NOT NULL,
    state_start INTEGER NOT NULL,
    state_end INTEGER NOT NULL
);`

type sequencesEntry struct {
	PartitionKey string `json:"partitionKey,omitempty"`
	ID           string `json:"id,omitempty"`
	Key          string `json:"key,omitempty"`
	PlanID       string `json:"planID,omitempty"`
	Name         string `json:"name,omitempty"`
	Descr        string `json:"descr,omitempty"`
	Pos          int64  `json:"pos,omitempty"`
	Actions      string `json:"actions,omitempty"`
	StateStatus  int64  `json:"stateStatus,omitempty"`
	StateStart   int64  `json:"stateStart,omitempty"`
	StateEnd     int64  `json:"stateEnd,omitempty"`
}

var sequencesSchema = `
CREATE Table If Not Exists sequences (
    id TEXT PRIMARY KEY,
    key TEXT,
    plan_id TEXT NOT NULL,
    name TEXT NOT NULL,
    descr TEXT NOT NULL,
    pos INTEGER NOT NULL,
    actions BLOB NOT NULL,
    state_status INTEGER NOT NULL,
    state_start INTEGER NOT NULL,
    state_end INTEGER NOT NULL
);`

type actionsEntry struct {
	PartitionKey string `json:"partitionKey,omitempty"`
	ID           string `json:"id,omitempty"`
	Key          string `json:"key,omitempty"`
	PlanID       string `json:"planID,omitempty"`
	Name         string `json:"name,omitempty"`
	Descr        string `json:"descr,omitempty"`
	Pos          int64  `json:"pos,omitempty"`
	Plugin       string `json:"plugin,omitempty"`
	Timeout      int64  `json:"timeout,omitempty"`
	Retries      int64  `json:"retries,omitempty"`
	Req          string `json:"req,omitempty"`
	Attempts     int64  `json:"attempts,omitempty"`
	StateStatus  int64  `json:"stateStatus,omitempty"`
	StateStart   int64  `json:"stateStart,omitempty"`
	StateEnd     int64  `json:"stateEnd,omitempty"`
}

var actionsSchema = `
CREATE Table If Not Exists actions (
    id TEXT PRIMARY KEY,
    key TEXT,
    plan_id TEXT NOT NULL,
    name TEXT NOT NULL,
    descr TEXT NOT NULL,
    pos INTEGER NOT NULL,
    plugin TEXT NOT NULL,
    timeout INTEGER NOT NULL,
    retries INTEGER NOT NULL,
    req BLOB,
    attempts BLOB,
    state_status INTEGER NOT NULL,
    state_start INTEGER NOT NULL,
    state_end INTEGER NOT NULL
);`

var indexes = []string{
	`CREATE INDEX If Not Exists idx_plans ON plans(id, group_id, state_status, state_start, state_end, reason);`,
	`CREATE INDEX If Not Exists idx_blocks ON blocks(id, key, plan_id, state_status, state_start, state_end);`,
	`CREATE INDEX If Not Exists idx_checks ON checks(id, key, plan_id, state_status, state_start, state_end);`,
	`CREATE INDEX If Not Exists idx_sequences ON sequences(id, key, plan_id, state_status, state_start, state_end);`,
	`CREATE INDEX If Not Exists idx_actions ON actions(id, key, plan_id, state_status, state_start, state_end, plugin);`,
}
