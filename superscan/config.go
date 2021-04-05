package superscan

//Scan add comment
type Scan struct {
	//需配置参数
	Ip           string `json:"ip"`
	PortRange    string `json:"port_range"`
	ActivePort   string `json:"active_port"`
	ThreadNumber int    `json:"thread_number"`

	//任务执行结果
	TimeOut int `json:"timeout"`
	//
	Callback    func(interface{}) `json:"-"`
	BarCallback func(i int)       `json:"-"`
}
