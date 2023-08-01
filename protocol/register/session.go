package register

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GetStoragePath 获取存储的路径
func (session *WaRegistration) GetStoragePath() string {
	if len(session.CachePath) == 0 {
		session.CachePath = "./cache/"
	}
	return fmt.Sprintf("%v%v%v", session.CachePath, session.WAId, "-session.json")
}

// ReadCache 读取JSON格式的session文件
func (session *WaRegistration) ReadCache() error {

	//获取存储路径
	path := session.GetStoragePath()

	//打开存储文件;
	sessFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer sessFile.Close()

	//解析存储数据;
	if err := json.NewDecoder(sessFile).Decode(session); err != nil {
		return err
	}

	return nil
}

// WriteCache 写入JSON格式的session文件
func (session *WaRegistration) WriteCache() error {
	//
	path := session.GetStoragePath()

	//创建目录
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return err
	}
	var sessFile *os.File
	_, err = os.Stat(path)
	//如果不存在
	if os.IsNotExist(err) {
		//创建存储文件;
		sessFile, err = os.Create(path)
		if err != nil {
			return err
		}
		defer sessFile.Close()
	} else {
		//创建存储文件;
		sessFile, err = os.Open(path)
		if err != nil {
			return err
		}
		defer sessFile.Close()
	}

	//编码写入配置文件;
	sessEncoder := json.NewEncoder(sessFile)
	//设置设置缩进一个制表符
	sessEncoder.SetIndent("", "\t")
	if err := sessEncoder.Encode(session); err != nil {
		return err
	}

	return nil

}

// RemoveCache 删除缓存文件
func (session *WaRegistration) RemoveCache() error {
	path := session.GetStoragePath()
	return os.Remove(path)
}
