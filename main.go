package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
	"github.com/parnurzeal/gorequest"
	"github.com/spf13/viper"
)

type Config struct {
	Database struct {
		Connection string
		Debug      bool
	}
}

type AreaResponse struct {
	ShowapiResCode   int64  `json:"showapi_res_code"`
	ShowapiResError  string `json:"showapi_res_error"`
	AreaResponseBody struct {
		RetCode interface{} `json:"ret_code"`
		Flag    bool        `json:"flag"`
		Data    []Area      `json:"data,omitempty"`
		Msg     string      `json:"msg,omitempty"`
	} `json:"showapi_res_body"`
}

type Area struct {
	Id         int64
	ProvinceId string `json:"provinceId"`
	SimpleName string `json:"simpleName"`
	Lon        string `json:"lon"`
	AreaCode   string `json:"areaCode"`
	CityId     string `json:"cityId"`
	Remark     string `json:"remark"`
	PrePinYin  string `json:"prePinYin"`
	Uid        string `json:"id"`
	PinYin     string `json:"pinYin"`
	ParentId   string `json:"parentId"`
	Level      int64  `json:"level"`
	AreaName   string `json:"areaName"`
	SimplePy   string `json:"simplePy"`
	ZipCode    string `json:"zipCode"`
	CountyId   string `json:"countyId"`
	Lat        string `json:"lat"`
	WholeName  string `json:"wholeName"`
}

var (
	x       *xorm.Engine
	record  map[string]int
	appCode string
	config  Config
)

func main() {
	// Get APPCODE here https://market.aliyun.com/products/57126001/cmapi011154.html
	appCode = os.Getenv("APPCODE")
	if appCode == "" {
		log.Fatalln("APPCODE not found")
	}
	// Load config
	loadConfig()

	// Init db
	var err error
	x, err = xorm.NewEngine("mysql", config.Database.Connection)
	if err != nil {
		log.Fatalln("Init DB error: ", err)
	}
	x.ShowSQL(config.Database.Debug)
	err = x.Sync2(new(Area))
	if err != nil {
		log.Fatalln("Sync DB error: ", err)
	}
	record = make(map[string]int)
	err = exportParentId("0")
	if err != nil {
		log.Println("Error export for id: 0, ", err)
	}
}

func exportParentId(parentId string) error {
	url := fmt.Sprintf("http://ali-city.showapi.com/areaDetail?parentId=%s", parentId)
	log.Println("Start request: ", url)
	resp, body, errs := gorequest.New().Get(url).Set("Authorization", "APPCODE "+appCode).EndBytes()
	if errs != nil {
		return errs[0]
	}
	if resp.StatusCode != 200 && resp.StatusCode != 555 {
		return errors.New("Status code: " + strconv.Itoa(resp.StatusCode))
	}

	// fmt.Println("Api response: ", resp.Body)

	areaResponse := AreaResponse{}
	if err := json.Unmarshal(body, &areaResponse); err != nil {
		log.Println("Unmarshal area error")
	}
	// fmt.Println("Area data items: ", areaResponse)
	if len(areaResponse.AreaResponseBody.Data) == 0 {
		return nil
	}

	// Save data
	for _, a := range areaResponse.AreaResponseBody.Data {
		// fmt.Println("Area data: ", a, &a)
		if err := a.Create(); err != nil {
			log.Println("Error export for uid: ", a.Uid, ", ", err)
		}
		// Avoid request same uid
		if v, ok := record[a.Uid]; ok {
			record[a.Uid] = v + 1
		} else {
			record[a.Uid] = 1
			if err := exportParentId(a.Uid); err != nil {
				return err
			}
		}
	}
	return nil
}

func loadConfig() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalln("Fatal load config file: ", err)
	}

	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalln("Fatal unmarshal config file: ", err)
	}
}

func (area *Area) Create() error {
	affected, err := x.Insert(area)
	if affected == 0 {
		return errors.New("No affected")
	}
	return err
}
