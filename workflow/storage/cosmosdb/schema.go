package cosmosdb

var tables = []string{
	planSchema,
	blocksSchema,
	checksSchema,
	sequencesSchema,
	actionsSchema,
}

type plansEntry struct {
	id             string `json:"id,omitempty"`
	groupID        string `json:"groupID,omitempty"`
	name           string `json:"name,omitempty"`
	descr          string `json:"descr,omitempty"`
	meta           []byte `json:"meta,omitempty"`
	bypassChecks   string `json:"bypassChecks,omitempty"`
	preChecks      string `json:"preChecks,omitempty"`
	postChecks     string `json:"postChecks,omitempty"`
	contChecks     string `json:"contChecks,omitempty"`
	deferredChecks string `json:"deferredChecks,omitempty"`
	blocks         string `json:"blocks,omitempty"`
	stateStatus    int64  `json:"stateStatus,omitempty"`
	stateStart     int64  `json:"stateStart,omitempty"`
	stateEnd       int64  `json:"stateEnd,omitempty"`
	submitTime     int64  `json:"submitTime,omitempty"`
	reason         int64  `json:"reason,omitempty"`
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
	id                string `json:"id,omitempty"`
	planID            string `json:"planID,omitempty"`
	key               string `json:"key,omitempty"`
	name              string `json:"name,omitempty"`
	descr             string `json:"descr,omitempty"`
	pos               int64  `json:"pos,omitempty"`
	entanceDelay      int64  `json:"entranceDelay,omitempty"`
	exitDelay         int64  `json:"exitDelay,omitempty"`
	bypassChecks      string `json:"bypassChecks,omitempty"`
	preChecks         string `json:"preChecks,omitempty"`
	postChecks        string `json:"postChecks,omitempty"`
	contChecks        string `json:"contChecks,omitempty"`
	deferredChecks    string `json:"deferredChecks,omitempty"`
	sequences         string `json:"sequences,omitempty"`
	toleratedFailures int64  `json:"toleratedFailures,omitempty"`
	stateStatus       int64  `json:"stateStatus,omitempty"`
	stateStart        int64  `json:"stateStart,omitempty"`
	stateEnd          int64  `json:"stateEnd,omitempty"`
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
	id          string `json:"id,omitempty"`
	key         string `json:"key,omitempty"`
	planID      string `json:"planID,omitempty"`
	actions     string `json:"actions,omitempty"`
	delay       int64  `json:"delay,omitempty"`
	stateStatus int64  `json:"stateStatus,omitempty"`
	stateStart  int64  `json:"stateStart,omitempty"`
	stateEnd    int64  `json:"stateEnd,omitempty"`
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
	id          string `json:"id,omitempty"`
	key         string `json:"key,omitempty"`
	planID      string `json:"planID,omitempty"`
	name        string `json:"name,omitempty"`
	descr       string `json:"descr,omitempty"`
	pos         int64  `json:"pos,omitempty"`
	actions     string `json:"actions,omitempty"`
	stateStatus int64  `json:"stateStatus,omitempty"`
	stateStart  int64  `json:"stateStart,omitempty"`
	stateEnd    int64  `json:"stateEnd,omitempty"`
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
	id          string `json:"id,omitempty"`
	key         string `json:"key,omitempty"`
	planID      string `json:"planID,omitempty"`
	name        string `json:"name,omitempty"`
	descr       string `json:"descr,omitempty"`
	pos         int64  `json:"pos,omitempty"`
	plugin      string `json:"plugin,omitempty"`
	timeout     int64  `json:"timeout,omitempty"`
	retries     int64  `json:"retries,omitempty"`
	req         string `json:"req,omitempty"`
	attempts    int64  `json:"attempts,omitempty"`
	stateStatus int64  `json:"stateStatus,omitempty"`
	stateStart  int64  `json:"stateStart,omitempty"`
	stateEnd    int64  `json:"stateEnd,omitempty"`
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
