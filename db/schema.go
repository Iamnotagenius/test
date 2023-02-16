package main

import (
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"

	"github.com/Iamnotagenius/test/db/service"
)

func InitDb(connOpts *pg.Options) (*pg.DB, error) {
	db := pg.Connect(connOpts)
	models := []interface{}{
		(*service.User)(nil),
	}

	for _, model := range models {
		err := db.Model(model).CreateTable(&orm.CreateTableOptions{
			Temp:        false,
			IfNotExists: true,
		})
		if err != nil {
			return nil, err
		}
	}
	return db, nil
}
