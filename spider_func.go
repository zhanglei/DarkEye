package main

import (
	"fmt"
	"github.com/therecipe/qt/widgets"
	"github.com/zsdevX/DarkEye/common"
	"github.com/zsdevX/DarkEye/spider"
	"github.com/zsdevX/DarkEye/ui"
	"strconv"
	"time"
)

func LoadSpider(mainWindow *ui.MainWindow) {
	mainWindow.Spider_deps.SetText(strconv.Itoa(mConfig.Spider.MaxDeps))
	mainWindow.Spider_url.SetText(mConfig.Spider.Url)
	mainWindow.Spider_node_url.SetText(mConfig.Spider.RequestMatchRule)
	mainWindow.Search_key.SetText(mConfig.Spider.SearchAPIKey)
	mainWindow.Search_query.SetText(mConfig.Spider.Query)

	logC, runCtl := logChannel(mainWindow.Spider_log)
	//Action
	mainWindow.Spider_start.ConnectClicked(func(bool) {
		//保存配置
		mConfig.Spider = spider.NewConfig()
		mConfig.Spider.MaxDeps, _ = strconv.Atoi(mainWindow.Spider_deps.Text())
		mConfig.Spider.Url = mainWindow.Spider_url.Text()
		mConfig.Spider.RequestMatchRule = mainWindow.Spider_node_url.Text()
		mConfig.Spider.Cookie = mainWindow.Spider_cookie.Text()
		mConfig.Spider.Query = mainWindow.Search_query.Text()
		mConfig.Spider.SearchAPIKey = mainWindow.Search_key.Text()
		mConfig.Spider.SearchEnable = mainWindow.Search_enable.IsChecked()

		mConfig.Spider.ErrChannel = logC
		if err := saveCfg(); err != nil {
			logC <- common.LogBuild("UI", "保存配置失败:"+err.Error(), common.FAULT)
			return
		}
		//启动流程
		mainWindow.Spider_start.SetEnabled(false)
		mainWindow.Spider_import_urls.SetEnabled(false)
		mainWindow.Spider_stop.SetEnabled(true)
		common.StartIt(&mConfig.Spider.Stop)
		go func() {
			mConfig.Spider.Run()
			mainWindow.Spider_start.SetEnabled(true)
			mainWindow.Spider_stop.SetEnabled(false)
			runCtl <- false
		}()
	})

	mainWindow.Spider_stop.ConnectClicked(func(bool) {
		common.StopIt(&mConfig.Spider.Stop)
		mainWindow.Spider_stop.SetDisabled(true)
		//异步处理等待结束避免界面卡顿
		go func() {
			sec := 0
			stop := false
			tick := time.NewTicker(time.Second)
			for {
				select {
				case <-runCtl:
					stop = true
				case <-tick.C:
					sec ++
					mainWindow.Spider_stop.SetText(fmt.Sprintf("等待%d秒", 60-sec))
				}
				if stop {
					break
				}
			}
			mainWindow.Spider_import_urls.SetEnabled(true)
			mainWindow.Spider_start.SetEnabled(true)
			mainWindow.Spider_stop.SetText("停止")
		}()
	})

	mainWindow.Spider_import_urls.ConnectClicked(func(bool) {
		qFile := widgets.NewQFileDialog2(nil, "选择url列表文件", ".", "")
		f := qFile.GetOpenFileName(nil, "文件", ".", "", "", widgets.QFileDialog__ReadOnly)
		if f == "" {
			return
		}
		defer qFile.Close()
		urls, err := common.ImportFiles(f, mainWindow.Spider_url.Text())
		if err != nil {
			logC <- common.LogBuild("UI", "加载文件失败"+err.Error(), common.ALERT)
			return
		}
		mainWindow.Spider_url.SetText(urls)
		logC <- common.LogBuild("UI", "批量导入完成", common.INFO)
	})
	return
}