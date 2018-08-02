package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"

	"github.com/Unknwon/goconfig"
	"github.com/otwdev/galaxylib"
)

type AliRQ struct {
	DataFile   string
	Account    string
	cookie     string
	param      string
	startTime  string
	endTime    string
	queryURL   string
	jar        *galaxylib.Jar
	alicookies []*http.Cookie
	ctoken     string
}

func NewAliRQ(file string) *AliRQ {
	return &AliRQ{
		DataFile: file,
	}
}

//var ctoken string

func (a *AliRQ) loadConfg() {

	fmt.Println(a.Account)

	cfg, err := goconfig.LoadConfigFile(a.DataFile)
	if err != nil {
		galaxylib.GalaxyLogger.Error(err)
		return
	}

	a.cookie = cfg.MustValue("rq", "cookie")
	a.param = cfg.MustValue("rq", "params")
	a.startTime = cfg.MustValue("rq", "startTime")
	a.endTime = cfg.MustValue("rq", "endTime")
	a.queryURL = cfg.MustValue("rq", "url")

}

func (a *AliRQ) RQ() {

	if a.jar == nil {
		a.jar = galaxylib.NewJar()
		a.loadConfg()
		a.alicookies = a.jar.BulkEdit(a.cookie)
	}

	startTime := time.Now().Format("2006-01-02 00:00:00")
	endTime := time.Now().Add(24 * time.Hour).Format("2006-01-02 00:00:00")

	if len(a.startTime) > 0 {
		startTime = a.startTime
	}

	if len(a.endTime) > 0 {
		endTime = a.endTime
	}

	params, _ := url.ParseQuery(a.param)

	params.Set("startDateInput", startTime)
	params.Set("endDateInput", endTime)

	for _, c := range a.alicookies {

		if c.Name == "ctoken" {
			params.Set("ctoken", c.Value)
		}
	}

	rq, _ := http.NewRequest("POST", a.queryURL, strings.NewReader(params.Encode()))

	section, _ := galaxylib.GalaxyCfgFile.GetSection("rqheader")

	for key, s := range section {
		rq.Header.Add(strings.TrimSpace(key), strings.TrimSpace(s))
	}

	a.jar.SetCookies(rq.URL, a.alicookies)
	c := http.Client{
		Jar: a.jar,
	}

	rs, err := c.Do(rq)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rs.Body.Close()

	buf, _ := ioutil.ReadAll(rs.Body)

	for _, c := range rs.Cookies() {
		has := false
		for i, acookie := range a.alicookies {
			if acookie.Name == strings.TrimSpace(c.Name) {
				a.alicookies[i].Value = c.Value
				has = true
			}
		}
		if !has {
			a.alicookies = append(a.alicookies, c)
		}
		if c.Name == "ctoken" {
			a.ctoken = c.Value
			fmt.Println(a.ctoken)
		}
	}

	buf, _ = simplifiedchinese.GB18030.NewDecoder().Bytes(buf)

	fmt.Println(galaxylib.DefaultGalaxyTools.Bytes2CHString(buf))

	var trade *AliTradeItem

	json.Unmarshal(buf, &trade)

	if trade == nil {
		//data := galaxylib.DefaultGalaxyTools.Bytes2CHString(buf)

		fmt.Println(string(buf))
		return
	}

	if trade.Stat == "deny" {
		galaxylib.GalaxyLogger.Warnln(fmt.Sprintf("账号：%s超时", a.Account))
		return
	}

	trade.Account = a.Account

	tradeBuf, _ := json.Marshal(trade)

	//galaxylib.DefaultGalaxyTools.Bytes2CHString(tradeBuf)
	//fmt.Println(string(tradeBuf))

	a.sendToAPI(tradeBuf)

	galaxylib.GalaxyLogger.Infoln(fmt.Sprintf("%s账号-数据：%d-查询条件:%s", a.Account, trade.Result.Summary.ExpendSum.Count, params.Encode()))

}

func (a *AliRQ) sendToAPI(t []byte) {

	apiURL := galaxylib.GalaxyCfgFile.MustValue("api", "url")
	res, err := http.Post(apiURL, "application/json", bytes.NewBuffer(t))
	if err != nil {
		galaxylib.GalaxyLogger.Errorln(err)
		return
	}
	defer res.Body.Close()

	//galaxylib.GalaxyLogger.Infoln(galaxylib.DefaultGalaxyTools.ResponseToString(res.Body))
}
