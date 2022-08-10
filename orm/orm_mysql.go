package orm

/**
 * @Author  糊涂的老知青
 * @Date    2021/10/30
 * @Version 1.0.0
 */

import (
	"errors"
	"fmt"
	token "lzq-admin/pkg/auth"
	"lzq-admin/pkg/hsflogger"
	"reflect"
	"time"

	"github.com/ahmetb/go-linq/v3"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"
	"xorm.io/xorm/names"
)

var DB *xorm.Engine

type LzqDatabase struct {
	Type        string `mapstructure:"type"`
	Host        string `mapstructure:"host"`
	UserName    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	Database    string `mapstructure:"database"`
	MaxOpenConn int    `mapstructure:"max_open_conn"`
	MaxIdleConn int    `mapstructure:"max_idle_conn"`
	IsMigration bool   `mapstructure:"is_migration"`
}

func NewDatabase(migrationModels []interface{}) {
	var 
	var err error
	// 拼接连接字符串
	DB, err = xorm.NewEngine(lzqdb.Type, fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8",
		lzqdb.UserName,
		lzqdb.Password,
		lzqdb.Host,
		lzqdb.Database))
	if err != nil {
		hsflogger.LogError("Mysql 数据库连接失败", err)
	}
	if lzqdb.Type == "mysql" {
		DB.StoreEngine("ENGINE=InnoDB")
		DB.Charset("utf8mb4")
	}
	DB.SetColumnMapper(names.SameMapper{})
	// 设置打开数据库连接的最大数量
	DB.SetMaxOpenConns(lzqdb.MaxOpenConn)
	// 设置空闲连接池中连接的最大数量
	DB.SetMaxIdleConns(lzqdb.MaxIdleConn)
	// 设置了连接可复用的最大时间
	DB.SetConnMaxLifetime(4 * time.Hour)
	// 打印SQL
	DB.ShowSQL(true)
	// 迁移数据库
	if lzqdb.IsMigration {
		// 自动迁移模式
		err := DB.Sync2(migrationModels...)
		if err != nil {
			fmt.Println(2, err)
		}
	}

}

func NewLzqOrm(c *gin.Context) *LzqOrm {
	return &LzqOrm{
		ginCtx: c,
	}
}

type LzqOrm struct {
	ginCtx *gin.Context
}

func (o *LzqOrm) BeginTrans() (*xorm.Session, error) {
	DbSession := DB.NewSession()
	defer DbSession.Close()
	return DbSession, DbSession.Begin()
}

// QSession 查询DB
func (o *LzqOrm) QSession(useMultiTenancy bool, tAlias ...string) *xorm.Session {
	queryDB := DB.Where("1=1")
	tableAlias := make([]string, 0)
	if len(tAlias) > 0 && tAlias[0] != "" {
		for _, v := range tAlias {
			tableAlias = append(tableAlias, v+".")
			queryDB.And(v+"."+"IsDeleted=?", 0)
		}
	} else {
		queryDB.And("IsDeleted=?", 0)
	}

	useMultiTenancy = getUseMultiTenancy(useMultiTenancy)
	if useMultiTenancy {
		tenantId := token.GetCurrentTenantId(o.ginCtx)
		if len(tableAlias) > 0 {
			for _, v := range tableAlias {
				queryDB.Where(v+"TenantId=?", tenantId)
			}

		} else {
			queryDB.Where("TenantId=?", tenantId)
		}
	}
	return queryDB
}

// ISession 插入DBSession
func (o *LzqOrm) ISession() *xorm.Session {
	claims := token.GetClaims(o.ginCtx)
	// 插入DB
	iBefore := func(obj interface{}) {
		beforeInsert(config.Config.ServerConfig.UseMultiTenancy, claims.TenantId, claims.Id, obj)
	}
	return DB.Before(iBefore)

}

// ISessionWithTrans 事务插入DBSession
func (o *LzqOrm) ISessionWithTrans(dbSession *xorm.Session) *xorm.Session {
	claims := token.GetClaims(o.ginCtx)
	// 插入DB
	iBefore := func(obj interface{}) {
		beforeInsert(config.Config.ServerConfig.UseMultiTenancy, claims.TenantId, claims.Id, obj)
	}
	return dbSession.Before(iBefore)

}

func (o *LzqOrm) InsertWithCreateId(objs []interface{}) []interface{} {
	claims := token.GetClaims(o.ginCtx)
	result := make([]interface{}, 0)
	for _, v := range objs {
		obj := beforeInsert(config.Config.ServerConfig.UseMultiTenancy, claims.TenantId, claims.Id, v)
		result = append(result, obj)
	}
	return result
}

func (o *LzqOrm) UpdateWithModityId(objs []interface{}) []interface{} {
	userId := token.GetCurrentUserId(o.ginCtx)
	result := make([]interface{}, 0)
	for _, v := range objs {
		obj, _, _ := beforeUpdate(userId, v)
		result = append(result, obj)
	}
	return result
}

// USession 修改DB
func (o *LzqOrm) USession(useMultiTenancy bool) *xorm.Session {
	isModityId := false
	isModityTime := false
	claims := token.GetClaims(o.ginCtx)
	// 修改DB
	uBefore := func(obj interface{}) {
		t := reflect.TypeOf(obj)
		if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
			_, isModityId, isModityTime = beforeUpdate(claims.Id, obj)
		}
	}
	updateDB := DB.Before(uBefore).Where("IsDeleted=?", 0)
	useMultiTenancy = getUseMultiTenancy(useMultiTenancy)
	if useMultiTenancy {
		updateDB.And("TenantId=?", claims.TenantId).Omit("TenantId")
	}
	if isModityId {
		updateDB.Cols("LastModifierId")
	}
	if isModityTime {
		updateDB.Cols("LastModificationTime")
	}
	return updateDB.Omit("Id")
}

// USessionWithTrans 事务修改DBSession
func (o *LzqOrm) USessionWithTrans(useMultiTenancy bool, dbSession *xorm.Session) *xorm.Session {
	// 修改DB
	isModityId := false
	isModityTime := false
	claims := token.GetClaims(o.ginCtx)
	uBefore := func(obj interface{}) {
		t := reflect.TypeOf(obj)
		if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
			_, isModityId, isModityTime = beforeUpdate(claims.Id, obj)
		}
	}
	dbSession.Before(uBefore).Where("IsDeleted=?", 0)
	useMultiTenancy = getUseMultiTenancy(useMultiTenancy)
	if useMultiTenancy {
		dbSession.And("TenantId=?", claims.TenantId).Omit("TenantId")
	}
	if isModityId {
		dbSession.Cols("LastModifierId")
	}
	if isModityTime {
		dbSession.Cols("LastModificationTime")
	}
	return dbSession.Omit("Id")
}

// DSession 删除DB
func (o *LzqOrm) DSession(useMultiTenancy bool) *xorm.Session {
	useMultiTenancy = getUseMultiTenancy(useMultiTenancy)
	isDeleterId := false
	isDeletionTime := false
	claims := token.GetClaims(o.ginCtx)
	// 删除DB
	dBefore := func(obj interface{}) {
		_, isDeleterId, isDeletionTime = beforeDelete(claims.Id, obj)
	}
	deleteDB := DB.Before(dBefore).UseBool("IsDeleted").Where("IsDeleted=?", 0)
	if isDeleterId {
		deleteDB.Cols("DeleterId")
	}
	if isDeletionTime {
		deleteDB.Cols("DeletionTime")
	}
	if useMultiTenancy {
		deleteDB.Where("TenantId=?", claims.TenantId).Omit("TenantId")
	}
	return deleteDB.Omit("Id")
}

// DSessionWithTrans 事务删除DBSession
func (o *LzqOrm) DSessionWithTrans(useMultiTenancy bool, dbSession *xorm.Session) *xorm.Session {
	useMultiTenancy = getUseMultiTenancy(useMultiTenancy)
	isDeleterId := false
	isDeletionTime := false
	claims := token.GetClaims(o.ginCtx)
	// 删除DB
	dBefore := func(obj interface{}) {
		_, isDeleterId, isDeletionTime = beforeDelete(claims.Id, obj)
	}
	deleteDB := dbSession.Before(dBefore).UseBool("IsDeleted").Where("IsDeleted=?", 0)
	if isDeleterId {
		deleteDB.Cols("DeleterId")
	}
	if isDeletionTime {
		deleteDB.Cols("DeletionTime")
	}
	if useMultiTenancy {
		deleteDB.Where("TenantId=?", claims.TenantId).Omit("TenantId")
	}
	return deleteDB.Omit("Id")
}

// 是否使用多租户
func getUseMultiTenancy(useMultiTenancy bool) bool {
	if config.Config.ServerConfig.UseMultiTenancy && useMultiTenancy {
		useMultiTenancy = true
	} else {
		useMultiTenancy = false
	}
	return useMultiTenancy
}

// GetUpdateFields 得到需要修改的字段
func GetUpdateFields(obj interface{}, omitFields ...string) ([]string, error) {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, errors.New("Check type error not Struct.")
	}
	omitFields = append(omitFields, "Id", "ID")
	fieldNum := t.NumField()
	var fieldNames = []string{"LastModificationTime", "LastModifierId"}
	for i := 0; i < fieldNum; i++ {
		if t.Field(i).Type.Kind() == reflect.Struct {
			if fieldC, err := GetUpdateFields(v.Field(i).Interface(), omitFields...); err != nil {
				return nil, err
			} else {
				fieldNames = append(fieldNames, fieldC...)
			}
		} else {
			fieldName := t.Field(i).Name
			isOmit := linq.From(omitFields).AnyWith(func(i interface{}) bool {
				return i == fieldName
			})
			if isOmit == false {
				fieldNames = append(fieldNames, fieldName)
			}
		}
	}
	return fieldNames, nil
}

// GetOptionFields 得到需要操作的字段
func GetOptionFields(obj interface{}, omitFields ...string) ([]string, error) {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, errors.New("Check type error not Struct.")
	}
	fieldNum := t.NumField()
	var fieldNames = make([]string, 0)
	for i := 0; i < fieldNum; i++ {
		if t.Field(i).Type.Kind() == reflect.Struct {
			if fieldC, err := GetUpdateFields(v.Field(i).Interface(), omitFields...); err != nil {
				return nil, err
			} else {
				fieldNames = append(fieldNames, fieldC...)
			}
		} else {
			fieldName := t.Field(i).Name
			isOmit := linq.From(omitFields).AnyWith(func(i interface{}) bool {
				return i == fieldName
			})
			if isOmit == false {
				fieldNames = append(fieldNames, fieldName)
			}
		}
	}
	return fieldNames, nil
}

func (o *LzqOrm) ConditionWithDeletedOrTenantId(useMultiTenancy bool, condition, tAlias string) string {
	condition = fmt.Sprintf("%v and %v.IsDeleted=%v", condition, tAlias, 0)
	useMultiTenancy = getUseMultiTenancy(useMultiTenancy)
	if useMultiTenancy {
		condition = fmt.Sprintf("%v and %v.TenantId=%v", condition, tAlias, token.GetCurrentTenantId(o.ginCtx))
	}
	return condition
}

func beforeInsert(useMultiTenancy bool, tenantId, userId string, obj interface{}) interface{} {
	immutable := reflect.ValueOf(obj).Elem()
	if (len(userId) > 0 && immutable.FieldByName("CreatorId") != reflect.Value{}) {
		immutable.FieldByName("CreatorId").SetString(userId)
	}
	if (useMultiTenancy && len(tenantId) > 0 && immutable.FieldByName("TenantId") != reflect.Value{}) {
		immutable.FieldByName("TenantId").SetString(tenantId)
	}
	return obj
}

func beforeUpdate(userId string, obj interface{}) (interface{}, bool, bool) {
	immutable := reflect.ValueOf(obj).Elem()
	isModityId := false
	isModityTime := false
	if (immutable.FieldByName("LastModificationTime") != reflect.Value{}) {
		isModityTime = true
		immutable.FieldByName("LastModificationTime").Set(reflect.ValueOf(time.Now()))
	}
	if (immutable.FieldByName("LastModifierId") != reflect.Value{}) {
		isModityId = true
		immutable.FieldByName("LastModifierId").SetString(userId)
	}
	return obj, isModityId, isModityTime
}

func beforeDelete(userId string, obj interface{}) (interface{}, bool, bool) {
	immutable := reflect.ValueOf(obj).Elem()
	isDeleterId := false
	isDeletionTime := false
	if (immutable.FieldByName("DeletionTime") != reflect.Value{}) {
		immutable.FieldByName("DeletionTime").Set(reflect.ValueOf(time.Now()))
		isDeletionTime = true
	}
	if (immutable.FieldByName("IsDeleted") != reflect.Value{}) {
		immutable.FieldByName("IsDeleted").SetBool(true)
	}
	if (immutable.FieldByName("DeleterId") != reflect.Value{}) {
		immutable.FieldByName("DeleterId").SetString(userId)
		isDeleterId = true
	}
	return obj, isDeleterId, isDeletionTime
}
