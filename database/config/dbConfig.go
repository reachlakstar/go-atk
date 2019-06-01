package config

import (
	"fmt"

	"github.com/micro/go-config"
	"github.com/micro/go-config/source/file"
)


// define our own host type
type DBConfig struct {
	Address string `json:"address"`
	Database string `json:"database"`
	UserName string `json:"username"`
	Password string `json:"password"`
}


// ReadConfig reads the config file
func ReadConfig(fileConfig string) (DBConfig, error) {

	// load the config from a file source
	if err := config.Load(file.NewSource(
		file.WithPath(fileConfig),
	)); err != nil {
		fmt.Println(err)
		return DBConfig{}, err
	}

	dbConfig := DBConfig{}

	// read a database host
	if err := config.Get("hosts", "database").Scan(&dbConfig); err != nil {
		fmt.Println(err)
		return DBConfig{}, err
	}

	fmt.Println(dbConfig.Address)

	return dbConfig, nil
}