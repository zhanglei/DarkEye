package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/noborus/trdsql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"sort"
)

type analysisEntity struct {
	ID      int64  `json:"id" gorm:"primaryKey"`
	Task    string `json:"task" gorm:"unique_index:UNIQ_hi;column:task"`
	Ip      string `json:"ip" gorm:"unique_index:UNIQ_hi;column:ip"`
	Port    string `json:"port" gorm:"unique_index:UNIQ_hi;column:port"`
	Service string `json:"service" gorm:"unique_index:UNIQ_hi;column:service"`

	Url             string `json:"url" gorm:"column:url"`
	Title           string `json:"title" gorm:"column:title"`
	WebServer       string `json:"web_server" gorm:"column:web_server"`
	WebResponseCode int32  `json:"http_code" gorm:"column:http_code"`

	Hostname  string
	Os        string
	Device    string
	Banner    string
	Version   string
	ExtraInfo string
	RDns      string
	Country   string

	NetBios     string `json:"netbios" gorm:"column:netbios"`
	WeakAccount string `json:"weak_account" gorm:"column:weak_account"`
	Vulnerable  string `json:"vulnerable" gorm:"column:vulnerable"`
}

func (analysisEntity) TableName() string {
	return "ent"
}

type analysisRuntime struct {
	d       *gorm.DB
	q       string
	flagSet *flag.FlagSet
}

var (
	analysisRuntimeOptions = &analysisRuntime{
		flagSet: flag.NewFlagSet("analysis", flag.ExitOnError),
	}
	analysisDb = analysisProgram + ".s3db"
)

func analysisInitRunTime() {
	analysisRuntimeOptions.flagSet.StringVar(&analysisRuntimeOptions.q, "sql", "select * from ent limit 1", "Sqlite3 Grammar")

	db, err := gorm.Open(sqlite.Open(analysisDb), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(err.Error())
	}
	analysisRuntimeOptions.d = db
	err = db.AutoMigrate(&analysisEntity{})
	if err != nil {
		panic(err.Error())
	}
}

func (a *analysisRuntime) compileArgs(cmd []string) error {
	if err := a.flagSet.Parse(splitCmd(cmd)); err != nil {
		return err
	}
	a.flagSet.Parsed()
	return nil
}

func (a *analysisRuntime) usage() {
	fmt.Println(fmt.Sprintf("Usage of %s:", analysisProgram))
	fmt.Println("Options:")
	a.flagSet.VisitAll(func(f *flag.Flag) {
		var opt = "  -" + f.Name
		fmt.Println(opt)
		fmt.Println(fmt.Sprintf("		%v (default '%v')", f.Usage, f.DefValue))
	})
}

func (a *analysisRuntime) start(ctx context.Context) {
	d := make([]analysisEntity, 0)
	ret := a.d.Raw(analysisRuntimeOptions.q).Scan(&d)
	if ret.Error != nil {
		return
	}

	sort.Slice(d, func(i, j int) bool {
		return d[j].Ip > d[i].Ip
	})

	fmt.Println("")
	jsonString, _ := json.Marshal(d)
	r := bytes.NewBuffer(jsonString)
	importer, err := trdsql.NewBufferImporter("ent", r, trdsql.InFormat(trdsql.JSON))
	if err != nil {
		fmt.Println("Err:", err.Error())
		return
	}
	writer := trdsql.NewWriter(trdsql.OutFormat(trdsql.AT))
	trd := trdsql.NewTRDSQL(importer, trdsql.NewExporter(writer))
	trd.Driver = "sqlite3"
	err = trd.Exec(a.q)
	if err != nil {
		fmt.Println("Err:", err.Error())
		return
	}
}

func (a *analysisRuntime) createOrUpdate(e *analysisEntity) {
	var n analysisEntity
	if a.d.Table("ent").Where(
		"task = ? and ip = ? and port = ? and service = ?",
		e.Task, e.Ip, e.Port, e.Service).First(&n).Error == gorm.ErrRecordNotFound {
		a.d.Create(e)
	} else {
		aJson, _ := json.Marshal(e)
		var m map[string]interface{}
		_ = json.Unmarshal(aJson, &m)
		for k, v := range m {
			switch v.(type) {
			case int32:
				if v.(int32) == 0 {
					delete(m, k)
				}
			case string:
				if v.(string) == "" {
					delete(m, k)
				}
			default:
				delete(m, k)
			}
		}
		a.d.Model(e).Where(
			"task = ? and ip = ? and port = ? and service = ?",
			e.Task, e.Ip, e.Port, e.Service).Updates(
				m)
	}
}