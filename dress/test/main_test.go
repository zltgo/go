package test

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zltgo/dress/conf"
	"github.com/zltgo/dress/graph"
	"github.com/zltgo/dress/model"
	gm "github.com/zltgo/webkit/graphorm"
	"github.com/zltgo/webkit/graphorm/client"
	"github.com/zltgo/webkit/jwt"
	"log"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"
)

var srv *httptest.Server
var cfg *conf.Conf
var m *model.Model
func init() {
	//delete db file
	//err := os.Remove("./test.sqlite3")
	//if err != nil {
	//	log.Println(err)
	//}

	//create new db
	cfg = conf.ReadCfg("../conf/conf.yaml")
	cfg.Model.Source = "./test.sqlite3?charset=utf8&parseTime=True&loc=Local"
	cfg.AuthOpts.HashKey = "1111111111111111"

	var err error
	m, err = model.NewModel(cfg.Model)
	if err != nil {
		log.Fatal(err)
	}

	runtime.SetFinalizer(m, func(m *model.Model) {
		m.CloseDB()
	})
	jwt.TimeNow = TimeFunc(1000, 1000)
	//add some records
	//mm := Model{m}
	//mm.AddAccount("马云", "15932323232", "淘宝tmall")
	//mm.AddAccount("马化腾", "15987653275", "微信wechat")
	//mm.AddAccount("李彦宏", "18609837575", "百度baidu")
	//
	//for i:= 0; i < 20; i++ {
	//	mm.AddAccount(fmt.Sprint("姓名", i), fmt.Sprint(15000000000+i), fmt.Sprint("公司名", i))
	//}

	srv = httptest.NewServer(NewServer(m, cfg))
}

func TestToken(t *testing.T) {
	c := client.New(srv.URL + "/graphql")
	t.Run("adminToken", func(t *testing.T) {
		var resp struct {
			AdminToken jwt.AuthToken
		}
		err := c.Post(`query {
				adminToken(name:"root", password:"112358") {
					accessToken
					refreshToken
					maxAge
				}
			}`,
			&resp)

		require.NoError(t, err)
		require.Equal(t, jwt.AuthToken{
			AccessToken:  "eyJUb2tlblZhbHVlcyI6eyJBZ2VudEhhc2giOiI4NWE0MGYyOTMyMGM0ZDlmIiwiR3JhbnRUeXBlIjoiYWNjZXNzIiwiTWV0YWRhdGEiOnsiRW1wbm8iOiJyb290IiwiUGFzc3dvcmRIYXNoIjoiYjdkYTQ1ZTI5NzVjM2YxMjk3MWUzYjNkMzlhNzI4ODMiLCJSb2xlIjoiYWRtaW4iLCJFeHBpcmVzQXQiOiIxOTcwLTAxLTAyVDA4OjE2OjQwLjAwMDAwMSswODowMCJ9fSwiRXhwaXJlc0F0Ijo0NjAwLCJJc3N1ZWRBdCI6MTAwMH0p1ixNltFkVkIrZvxYtTQZOiDpTA==",
			RefreshToken: "eyJUb2tlblZhbHVlcyI6eyJBZ2VudEhhc2giOiI4NWE0MGYyOTMyMGM0ZDlmIiwiR3JhbnRUeXBlIjoicmVmcmVzaCIsIk1ldGFkYXRhIjp7IkVtcG5vIjoicm9vdCIsIlBhc3N3b3JkSGFzaCI6ImI3ZGE0NWUyOTc1YzNmMTI5NzFlM2IzZDM5YTcyODgzIiwiUm9sZSI6ImFkbWluIiwiRXhwaXJlc0F0IjoiMTk3MC0wMS0wMlQwODoxNjo0MC4wMDAwMDErMDg6MDAifX0sIklzc3VlZEF0IjoxMDAwfY4n1TdWq1cCRmclFPjgIdAA1tuP",
			MaxAge:       cfg.AuthOpts.AccessMaxAge,
		}, resp.AdminToken)
	})

	t.Run("authToken", func(t *testing.T) {
		var resp struct {
			AccountID gm.UUID
		}
		err := c.Post(`query {
				accountId(mobile:"18609837575")
			}`,
			&resp)

		require.NoError(t, err)
		require.Equal(t, "5d6e1b8de138231520eb837f", resp.AccountID.String())

		//auth
		var tk struct {
			AuthToken jwt.AuthToken
		}
		err = c.Post(`query($au: AuthForm!){
					authToken(input:$au){
						accessToken
						refreshToken
						maxAge
					}
				}`,
			&tk,
			client.Var("au", model.AuthForm{
				AccountID: resp.AccountID,
				Empno:     "000",
				Password:  "000000000",
			}))

		require.NoError(t, err)
		require.Equal(t, cfg.AuthOpts.AccessMaxAge, tk.AuthToken.MaxAge)
		require.Equal(t, "eyJUb2tlblZhbHVlcyI6eyJBZ2VudEhhc2giOiI4NWE0MGYyOTMyMGM0ZDlmIiwiR3JhbnRUeXBlIjoiYWNjZXNzIiwiTWV0YWRhdGEiOnsiVXNlcklkIjoiNWQ2ZTFiOGRlMTM4MjMxNTIwZWI4MzgwIiwiQWNjb3VudElEIjoiNWQ2ZTFiOGRlMTM4MjMxNTIwZWI4MzdmIiwiRW1wbm8iOiIwMDAiLCJQYXNzd29yZEhhc2giOiJhOWQzZTgzOTcyMmM3NjAxYmQ5Y2E3NTdlMzk1YmU2ZiIsIlJvbGUiOiJtYW5hZ2VyIiwiRXhwaXJlc0F0IjoiMTk3MC0wMS0wMlQwODoxNjo0MC4wMDAwMDErMDg6MDAifX0sIkV4cGlyZXNBdCI6NDYwMCwiSXNzdWVkQXQiOjEwMDB9o_Y9PXjjAXiaGTvdN6OrkRxsGx0=", tk.AuthToken.AccessToken)
		require.Equal(t, "eyJUb2tlblZhbHVlcyI6eyJBZ2VudEhhc2giOiI4NWE0MGYyOTMyMGM0ZDlmIiwiR3JhbnRUeXBlIjoicmVmcmVzaCIsIk1ldGFkYXRhIjp7IlVzZXJJZCI6IjVkNmUxYjhkZTEzODIzMTUyMGViODM4MCIsIkFjY291bnRJRCI6IjVkNmUxYjhkZTEzODIzMTUyMGViODM3ZiIsIkVtcG5vIjoiMDAwIiwiUGFzc3dvcmRIYXNoIjoiYTlkM2U4Mzk3MjJjNzYwMWJkOWNhNzU3ZTM5NWJlNmYiLCJSb2xlIjoibWFuYWdlciIsIkV4cGlyZXNBdCI6IjE5NzAtMDEtMDJUMDg6MTY6NDAuMDAwMDAxKzA4OjAwIn19LCJJc3N1ZWRBdCI6MTAwMH0Qqw9j9lDE7qz6YCHT_6vbRnVwrA==", tk.AuthToken.RefreshToken)
	})

	t.Run("refreshToken", func(t *testing.T) {
		c := client.New(srv.URL + "/graphql")
		var resp struct {
			RefreshToken jwt.AuthToken
		}
		err := c.Post(`query {
				refreshToken {
					accessToken
					refreshToken
					maxAge
				}
			}`,
			&resp, func(r *client.Request) {
				r.Header.Set("REFRESH-TOKEN", "eyJUb2tlblZhbHVlcyI6eyJBZ2VudEhhc2giOiI4NWE0MGYyOTMyMGM0ZDlmIiwiR3JhbnRUeXBlIjoicmVmcmVzaCIsIk1ldGFkYXRhIjp7IlVzZXJJZCI6IjVkNmUxYjhkZTEzODIzMTUyMGViODM4MCIsIkFjY291bnRJRCI6IjVkNmUxYjhkZTEzODIzMTUyMGViODM3ZiIsIkVtcG5vIjoiMDAwIiwiUGFzc3dvcmRIYXNoIjoiYTlkM2U4Mzk3MjJjNzYwMWJkOWNhNzU3ZTM5NWJlNmYiLCJSb2xlIjoibWFuYWdlciIsIkV4cGlyZXNBdCI6IjE5NzAtMDEtMDJUMDg6MTY6NDAuMDAwMDAxKzA4OjAwIn19LCJJc3N1ZWRBdCI6MTAwMH0Qqw9j9lDE7qz6YCHT_6vbRnVwrA==")
			})

		require.NoError(t, err)
		require.Equal(t, jwt.AuthToken{
			AccessToken:  "eyJUb2tlblZhbHVlcyI6eyJBZ2VudEhhc2giOiI4NWE0MGYyOTMyMGM0ZDlmIiwiR3JhbnRUeXBlIjoiYWNjZXNzIiwiTWV0YWRhdGEiOnsiVXNlcklkIjoiNWQ2ZTFiOGRlMTM4MjMxNTIwZWI4MzgwIiwiQWNjb3VudElEIjoiNWQ2ZTFiOGRlMTM4MjMxNTIwZWI4MzdmIiwiRW1wbm8iOiIwMDAiLCJQYXNzd29yZEhhc2giOiJhOWQzZTgzOTcyMmM3NjAxYmQ5Y2E3NTdlMzk1YmU2ZiIsIlJvbGUiOiJtYW5hZ2VyIiwiRXhwaXJlc0F0IjoiMTk3MC0wMS0wMlQwODoxNjo0MC4wMDAwMDErMDg6MDAifX0sIkV4cGlyZXNBdCI6NDYwMCwiSXNzdWVkQXQiOjEwMDB9o_Y9PXjjAXiaGTvdN6OrkRxsGx0=",
			RefreshToken: "eyJUb2tlblZhbHVlcyI6eyJBZ2VudEhhc2giOiI4NWE0MGYyOTMyMGM0ZDlmIiwiR3JhbnRUeXBlIjoicmVmcmVzaCIsIk1ldGFkYXRhIjp7IlVzZXJJZCI6IjVkNmUxYjhkZTEzODIzMTUyMGViODM4MCIsIkFjY291bnRJRCI6IjVkNmUxYjhkZTEzODIzMTUyMGViODM3ZiIsIkVtcG5vIjoiMDAwIiwiUGFzc3dvcmRIYXNoIjoiYTlkM2U4Mzk3MjJjNzYwMWJkOWNhNzU3ZTM5NWJlNmYiLCJSb2xlIjoibWFuYWdlciIsIkV4cGlyZXNBdCI6IjE5NzAtMDEtMDJUMDg6MTY6NDAuMDAwMDAxKzA4OjAwIn19LCJJc3N1ZWRBdCI6MTAwMH0Qqw9j9lDE7qz6YCHT_6vbRnVwrA==",
			MaxAge:       cfg.AuthOpts.AccessMaxAge,
		}, resp.RefreshToken)
	})
}



type Account struct {
	gm.Model         `json:",squash"`
	Corp             string    `json:"corp" gorm:"type:varchar(128);not null"`
	ManagerID        gm.UUID   `json:"managerID" gorm:"type:char(12);not null"`
	Mobile           string    `json:"mobile" gorm:"type:char(11);unique_index;not null"`
	MaxStores        int       `json:"maxStores" gorm:"not null"`
	MaxUsersPerStore int       `json:"maxUsersPerStore" gorm:"not null"`
	Remarks          string    `json:"remarks" gorm:"type:varchar(255)"`
	ExpiresAt        time.Time `json:"expiresAt" gorm:"not null"`

	//belongs to, the tag does not work in related, use association("manager") instead
	Manager *User `json:"manager" gorm:"ForeignKey:ManagerID"`

	//has many
	Stores []Store `json:"stores" gorm:"ForeignKey:AccountID;save_associations:false"`
}

type Store struct {
	gm.Model  `json:",squash"`
	AccountID gm.UUID `json:"accountID" gorm:"type:char(12) REFERENCES accounts(id) on update no action on delete no action;unique_index:idx_acc_name;not null"`
	Name      string  `json:"name" gorm:"type:varchar(128);unique_index:idx_acc_name;not null"`
	Remarks   string  `json:"remarks" gorm:"type:varchar(255)"`
	//has many
	Users []User `json:"users" gorm:"ForeignKey:StoreID;save_associations:false"`
}

type User struct {
	gm.Model     `json:",squash"`
	AccountID    gm.UUID  `json:"accountID" gorm:"type:char(12);unique_index:idx_acc_emp;not null"`
	StoreID      *gm.UUID `json:"storeID" gorm:"type:char(12) REFERENCES stores(id) on update no action on delete no action"`
	Empno        string   `json:"empno" gorm:"type:char(12);unique_index:idx_acc_emp;not null"`
	PasswordHash gm.MD5   `json:"-" gorm:"type:char(16);not null"`
	Salt         string   `json:"-" gorm:"type:char(16);not null"`
	RealName     string   `json:"realName" gorm:"type:varchar(50);not null"`
	Role         string   `json:"role" gorm:"type:varchar(50);not null"`
}

func TestQurey(t *testing.T) {
	c := client.New(srv.URL + "/graphql")
	t.Run("account", func(t *testing.T) {
		//auth
		var resp struct {
			Account Account
		}
		err := c.Post(`query{
					account(id:"5d6e1b8de138231520eb837f"){
						id
  						createdAt
  						updatedAt
  						deletedAt
  						corp
  						managerId
  						mobile
  						maxStores
  						maxUsersPerStore
  						remarks
  						expiresAt
  						manager{
						    id
						    createdAt
  						    updatedAt
  						    deletedAt
                            accountId
                            empno
                            realName
                            role
                            storeId
						}
						stores{
 							id
						    createdAt
  						    updatedAt
  						    deletedAt
                            accountId
 						 	name
  							remarks
  							users{
								id
						    	createdAt
  						    	updatedAt
  						    	deletedAt
                            	accountId
                            	empno
                            	realName
                            	role
                            	storeId
							}
						}
					}
				}`,
			&resp, func(r *client.Request) {
				r.Header.Set("ACCESS-TOKEN", "eyJUb2tlblZhbHVlcyI6eyJBZ2VudEhhc2giOiI4NWE0MGYyOTMyMGM0ZDlmIiwiR3JhbnRUeXBlIjoiYWNjZXNzIiwiTWV0YWRhdGEiOnsiRW1wbm8iOiJyb290IiwiUGFzc3dvcmRIYXNoIjoiYjdkYTQ1ZTI5NzVjM2YxMjk3MWUzYjNkMzlhNzI4ODMiLCJSb2xlIjoiYWRtaW4iLCJFeHBpcmVzQXQiOiIxOTcwLTAxLTAyVDA4OjE2OjQwLjAwMDAwMSswODowMCJ9fSwiRXhwaXJlc0F0Ijo0NjAwLCJJc3N1ZWRBdCI6MTAwMH0p1ixNltFkVkIrZvxYtTQZOiDpTA==")
			})
		require.NoError(t, err)
		b, _ := json.MarshalIndent(resp.Account, "", "    ")
		t.Log(string(b))

		ac := resp.Account
		require.Equal(t, "百度baidu", ac.Corp)
		require.Equal(t, "18609837575", ac.Mobile)
		require.Equal(t, "公司：百度baidu, 手机：18609837575", ac.Remarks)
		require.Equal(t, 10, ac.MaxStores)
		require.Equal(t, 100, ac.MaxUsersPerStore)
		require.NotNil(t, ac.Manager)
		require.Equal(t, ac.ManagerID, ac.Manager.ID)
		require.Equal(t, ac.ID, ac.Manager.AccountID)
		require.Equal(t, "李彦宏", ac.Manager.RealName)
		require.Equal(t, "000", ac.Manager.Empno)
		require.Equal(t, model.ManagerRole, ac.Manager.Role)

		require.Len(t, ac.Stores, 3)
		for i := 0; i < 3; i++ {
			st := ac.Stores[i]
			require.Equal(t, ac.ID, st.AccountID)
			require.Equal(t, fmt.Sprint("分店", i), st.Name)
			require.Equal(t, fmt.Sprint("分店名：分店", i), st.Remarks)
			require.Len(t, st.Users, 3)

			for j := 0; j < 3; j++ {
				require.Equal(t, st.Users[j].Empno, fmt.Sprint(i,"00", j+1))
				require.Equal(t, st.Users[j].RealName, fmt.Sprint("张", j))
				require.Equal(t, st.Users[j].Role, fmt.Sprint("级别", j))
				require.Equal(t, st.Users[j].StoreID.String(), st.ID.String())
				require.Equal(t, st.Users[j].AccountID, ac.ID)
			}
		}
	})

	t.Run("user", func(t *testing.T) {
		//auth
		var resp struct {
			User struct{
				gm.Model     `json:",squash"`
				AccountID    gm.UUID  `json:"accountID" gorm:"type:char(12);unique_index:idx_acc_emp;not null"`
				StoreID      *gm.UUID `json:"storeID" gorm:"type:char(12) REFERENCES stores(id) on update no action on delete no action"`
				Empno        string   `json:"empno" gorm:"type:char(12);unique_index:idx_acc_emp;not null"`
				PasswordHash gm.MD5   `json:"-" gorm:"type:char(16);not null"`
				Salt         string   `json:"-" gorm:"type:char(16);not null"`
				RealName     string   `json:"realName" gorm:"type:varchar(50);not null"`
				Role         string   `json:"role" gorm:"type:varchar(50);not null"`
				Store		*model.Store `json:"store"`
				Account     struct{
					ID gm.UUID `json:"id"`
				}
			}
		}
		err := c.Post(`query{
					user(id:"5d6e1b8de138231520eb8380"){
						id
						createdAt
  						updatedAt
  						deletedAt
                       accountId
                       empno
                       realName
                       role
                       storeId
						store{
 							id
						}
						account{
 							id
						}
					}
				}`,
			&resp, func(r *client.Request) {
				r.Header.Set("ACCESS-TOKEN", "eyJUb2tlblZhbHVlcyI6eyJBZ2VudEhhc2giOiI4NWE0MGYyOTMyMGM0ZDlmIiwiR3JhbnRUeXBlIjoiYWNjZXNzIiwiTWV0YWRhdGEiOnsiRW1wbm8iOiJyb290IiwiUGFzc3dvcmRIYXNoIjoiYjdkYTQ1ZTI5NzVjM2YxMjk3MWUzYjNkMzlhNzI4ODMiLCJSb2xlIjoiYWRtaW4iLCJFeHBpcmVzQXQiOiIxOTcwLTAxLTAyVDA4OjE2OjQwLjAwMDAwMSswODowMCJ9fSwiRXhwaXJlc0F0Ijo0NjAwLCJJc3N1ZWRBdCI6MTAwMH0p1ixNltFkVkIrZvxYtTQZOiDpTA==")
			})
		require.NoError(t, err)
		require.Equal(t, resp.User.Empno, "000")
		require.Equal(t, resp.User.RealName, "李彦宏")
		require.Equal(t, resp.User.Role, model.ManagerRole)
		require.Nil(t, resp.User.Store)
		require.Equal(t, resp.User.AccountID.String(), "5d6e1b8de138231520eb837f")
		require.Equal(t, resp.User.Account.ID.String(), "5d6e1b8de138231520eb837f")
	})

	t.Run("store", func(t *testing.T) {
		//auth
		var resp struct {
			Store Store
		}
		err := c.Post(`query{
					store(id:"5d6e1b8de138231520eb8381"){
						id
						createdAt
  						updatedAt
  						deletedAt
                        accountId
 						name
  						remarks
  						users{
							id
						    createdAt
  						    updatedAt
  						    deletedAt
                            accountId
                            empno
                            realName
                            role
                            storeId
						}
					}
				}`,
			&resp, func(r *client.Request) {
				r.Header.Set("ACCESS-TOKEN", "eyJUb2tlblZhbHVlcyI6eyJBZ2VudEhhc2giOiI4NWE0MGYyOTMyMGM0ZDlmIiwiR3JhbnRUeXBlIjoiYWNjZXNzIiwiTWV0YWRhdGEiOnsiRW1wbm8iOiJyb290IiwiUGFzc3dvcmRIYXNoIjoiYjdkYTQ1ZTI5NzVjM2YxMjk3MWUzYjNkMzlhNzI4ODMiLCJSb2xlIjoiYWRtaW4iLCJFeHBpcmVzQXQiOiIxOTcwLTAxLTAyVDA4OjE2OjQwLjAwMDAwMSswODowMCJ9fSwiRXhwaXJlc0F0Ijo0NjAwLCJJc3N1ZWRBdCI6MTAwMH0p1ixNltFkVkIrZvxYtTQZOiDpTA==")
			})
		require.NoError(t, err)
		require.Equal(t, resp.Store.AccountID.String(), "5d6e1b8de138231520eb837f")
		require.Equal(t, resp.Store.ID.String(), "5d6e1b8de138231520eb8381")
		require.Equal(t, "分店0", resp.Store.Name)
		require.Equal(t, "分店名：分店0", resp.Store.Remarks)
		require.Len(t, resp.Store.Users, 3)

		for j := 0; j < 3; j++ {
			require.Equal(t, resp.Store.Users[j].Empno, fmt.Sprint("000", j+1))
			require.Equal(t, resp.Store.Users[j].RealName, fmt.Sprint("张", j))
			require.Equal(t, resp.Store.Users[j].Role, fmt.Sprint("级别", j))
			require.Equal(t, resp.Store.Users[j].StoreID.String(), resp.Store.ID.String())
			require.Equal(t, resp.Store.Users[j].AccountID, resp.Store.AccountID)
		}
	})


	t.Run("accountsConnection", func(t *testing.T) {
		//auth
		var resp struct {
			AccountsConnection struct{
				TotalCount int	`json:"totalCount"`
				Edges []graph.AccountsEdge	`json:"edges"`
				Accounts []model.Account	`json:"accounts"`
				PageInfo graph.PageInfo	`json:"pageInfo"`
			}
		}
		err := c.Post(`query searchAccounts($flt: AccountFilter){
					accountsConnection(filter: $flt, first: 10, after: null){
						totalCount
						edges {
							cursor
							node {
								id
								corp
							}
						}
  						accounts {
							id
							mobile
						}
  						pageInfo {
							startCursor
							endCursor
							hasNextPage
						}
					}
				}`,
			&resp,
			client.Var("flt", model.AccountFilter{
				Orders:[]string{"mobile"},
				Mobile:"159",
			}),
			func(r *client.Request) {
				r.Header.Set("ACCESS-TOKEN", "eyJUb2tlblZhbHVlcyI6eyJBZ2VudEhhc2giOiI4NWE0MGYyOTMyMGM0ZDlmIiwiR3JhbnRUeXBlIjoiYWNjZXNzIiwiTWV0YWRhdGEiOnsiRW1wbm8iOiJyb290IiwiUGFzc3dvcmRIYXNoIjoiYjdkYTQ1ZTI5NzVjM2YxMjk3MWUzYjNkMzlhNzI4ODMiLCJSb2xlIjoiYWRtaW4iLCJFeHBpcmVzQXQiOiIxOTcwLTAxLTAyVDA4OjE2OjQwLjAwMDAwMSswODowMCJ9fSwiRXhwaXJlc0F0Ijo0NjAwLCJJc3N1ZWRBdCI6MTAwMH0p1ixNltFkVkIrZvxYtTQZOiDpTA==")
			})

		require.NoError(t, err)
		ac := &resp.AccountsConnection
		require.Equal(t, ac.TotalCount, 2)
		require.Len(t, ac.Accounts, 2)
		require.Equal(t, ac.Accounts[0].Mobile, "15932323232")
		require.Equal(t, ac.Accounts[1].Mobile, "15987653275")

		require.Len(t, ac.Edges, 2)
		require.Equal(t, ac.Edges[0].Cursor.String(), "5d6e1b8de138231520eb8363")
		require.Equal(t, ac.Edges[1].Cursor.String(), "5d6e1b8de138231520eb8371")


		require.Equal(t, ac.PageInfo.StartCursor.String(), "5d6e1b8de138231520eb8363")
		require.Equal(t, ac.PageInfo.EndCursor.String(), "5d6e1b8de138231520eb8371")
		require.False(t, ac.PageInfo.HasNextPage)
	})
}


func TestMutation(t *testing.T) {
	c := client.New(srv.URL + "/graphql")

	t.Run("newAccount", func(t *testing.T) {
		//newAccount
		var resp struct {
			NewAccount model.Account
		}

		err := c.Post(`mutation($nc: NewAccount!){
					newAccount(input: $nc){
						id
						mobile
					}
				}`,
			&resp,
			client.Var("nc", model.NewAccount{
				Empno:            "000",
				Password:         "000000000",
				RealName:         "测试一下",
				Mobile:           "13500009537",
				Corp:             "测试CreateAccount",
				MaxStores:        2,
				MaxUsersPerStore: 2,
				ExpiresAt:        time.Now().Add(-time.Hour),
			}),
			func(r *client.Request) {
				r.Header.Set("ACCESS-TOKEN", "eyJUb2tlblZhbHVlcyI6eyJBZ2VudEhhc2giOiI4NWE0MGYyOTMyMGM0ZDlmIiwiR3JhbnRUeXBlIjoiYWNjZXNzIiwiTWV0YWRhdGEiOnsiRW1wbm8iOiJyb290IiwiUGFzc3dvcmRIYXNoIjoiYjdkYTQ1ZTI5NzVjM2YxMjk3MWUzYjNkMzlhNzI4ODMiLCJSb2xlIjoiYWRtaW4iLCJFeHBpcmVzQXQiOiIxOTcwLTAxLTAyVDA4OjE2OjQwLjAwMDAwMSswODowMCJ9fSwiRXhwaXJlc0F0Ijo0NjAwLCJJc3N1ZWRBdCI6MTAwMH0p1ixNltFkVkIrZvxYtTQZOiDpTA==")
			})
		assert.NoError(t, err)
		assert.Equal(t, resp.NewAccount.Mobile, "13500009537")
	})

	t.Run("enable and disable", func(t *testing.T) {
		//disable account
		var res struct {
			DisableAccount bool
		}
		err := c.Post(`mutation($id: ID!){
					disableAccount(accountId: $id)
				}`,
			&res,
			client.Var("id", gm.UUID("5d6e226ae1382327105d781e")),
			func(r *client.Request) {
				r.Header.Set("ACCESS-TOKEN", "eyJUb2tlblZhbHVlcyI6eyJBZ2VudEhhc2giOiI4NWE0MGYyOTMyMGM0ZDlmIiwiR3JhbnRUeXBlIjoiYWNjZXNzIiwiTWV0YWRhdGEiOnsiRW1wbm8iOiJyb290IiwiUGFzc3dvcmRIYXNoIjoiYjdkYTQ1ZTI5NzVjM2YxMjk3MWUzYjNkMzlhNzI4ODMiLCJSb2xlIjoiYWRtaW4iLCJFeHBpcmVzQXQiOiIxOTcwLTAxLTAyVDA4OjE2OjQwLjAwMDAwMSswODowMCJ9fSwiRXhwaXJlc0F0Ijo0NjAwLCJJc3N1ZWRBdCI6MTAwMH0p1ixNltFkVkIrZvxYtTQZOiDpTA==")
			})
		require.NoError(t, err)
		require.True(t, res.DisableAccount)

		//enable
		var resp struct {
			EnableAccount bool
		}
		err = c.Post(`mutation($id: ID!){
					enableAccount(accountId: $id)
				}`,
			&resp,
			client.Var("id", gm.UUID("5d6e226ae1382327105d781e")),
			func(r *client.Request) {
				r.Header.Set("ACCESS-TOKEN", "eyJUb2tlblZhbHVlcyI6eyJBZ2VudEhhc2giOiI4NWE0MGYyOTMyMGM0ZDlmIiwiR3JhbnRUeXBlIjoiYWNjZXNzIiwiTWV0YWRhdGEiOnsiRW1wbm8iOiJyb290IiwiUGFzc3dvcmRIYXNoIjoiYjdkYTQ1ZTI5NzVjM2YxMjk3MWUzYjNkMzlhNzI4ODMiLCJSb2xlIjoiYWRtaW4iLCJFeHBpcmVzQXQiOiIxOTcwLTAxLTAyVDA4OjE2OjQwLjAwMDAwMSswODowMCJ9fSwiRXhwaXJlc0F0Ijo0NjAwLCJJc3N1ZWRBdCI6MTAwMH0p1ixNltFkVkIrZvxYtTQZOiDpTA==")
			})
		require.NoError(t, err)
		require.True(t, resp.EnableAccount)
	})

	t.Run("modAccount", func(t *testing.T) {
		//newAccount
		var resp struct {
			ModAccount bool
		}

		err := c.Post(`mutation($nc: ModAccount!){
					modAccount(input: $nc)
				}`,
			&resp,
			client.Var("nc", model.ModAccount{
				ID:               "5d6e226ae1382327105d781e",
				Mobile:           "13500009550",
				Corp:             "新网易wy",
				MaxStores:        11,
				MaxUsersPerStore: 101,
				ExpiresAt:        time.Now().Add(time.Hour),
				Remarks:          "测试改备注",
			}),
			func(r *client.Request) {
				r.Header.Set("ACCESS-TOKEN", "eyJUb2tlblZhbHVlcyI6eyJBZ2VudEhhc2giOiI4NWE0MGYyOTMyMGM0ZDlmIiwiR3JhbnRUeXBlIjoiYWNjZXNzIiwiTWV0YWRhdGEiOnsiRW1wbm8iOiJyb290IiwiUGFzc3dvcmRIYXNoIjoiYjdkYTQ1ZTI5NzVjM2YxMjk3MWUzYjNkMzlhNzI4ODMiLCJSb2xlIjoiYWRtaW4iLCJFeHBpcmVzQXQiOiIxOTcwLTAxLTAyVDA4OjE2OjQwLjAwMDAwMSswODowMCJ9fSwiRXhwaXJlc0F0Ijo0NjAwLCJJc3N1ZWRBdCI6MTAwMH0p1ixNltFkVkIrZvxYtTQZOiDpTA==")
			})
		assert.NoError(t, err)
		assert.True(t, resp.ModAccount)
	})
}


func BenchmarkQueryAccount(b *testing.B) {
	gin.SetMode(gin.TestMode)
	c := client.New(srv.URL + "/graphql")
	var resp struct {
		Account Account
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		//auth
		err := c.Post(`query{
					account(id:"5d6e1b8de138231520eb837f"){
						id
  						createdAt
  						updatedAt
  						deletedAt
  						corp
  						managerId
  						mobile
  						maxStores
  						maxUsersPerStore
  						remarks
  						expiresAt
  						manager{
						    id
						    createdAt
  						    updatedAt
  						    deletedAt
                            accountId
                            empno
                            realName
                            role
                            storeId
						}
						stores{
 							id
						    createdAt
  						    updatedAt
  						    deletedAt
                            accountId
 						 	name
  							remarks
  							users{
								id
						    	createdAt
  						    	updatedAt
  						    	deletedAt
                            	accountId
                            	empno
                            	realName
                            	role
                            	storeId
							}
						}
					}
				}`,
			&resp, func(r *client.Request) {
				r.Header.Set("ACCESS-TOKEN", "eyJUb2tlblZhbHVlcyI6eyJBZ2VudEhhc2giOiI4NWE0MGYyOTMyMGM0ZDlmIiwiR3JhbnRUeXBlIjoiYWNjZXNzIiwiTWV0YWRhdGEiOnsiRW1wbm8iOiJyb290IiwiUGFzc3dvcmRIYXNoIjoiYjdkYTQ1ZTI5NzVjM2YxMjk3MWUzYjNkMzlhNzI4ODMiLCJSb2xlIjoiYWRtaW4iLCJFeHBpcmVzQXQiOiIxOTcwLTAxLTAyVDA4OjE2OjQwLjAwMDAwMSswODowMCJ9fSwiRXhwaXJlc0F0Ijo0NjAwLCJJc3N1ZWRBdCI6MTAwMH0p1ixNltFkVkIrZvxYtTQZOiDpTA==")
			}, client.RandRemoteAddr)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.Log(m.CacheStatus())
}


func BenchmarkQueryUser(b *testing.B) {
	gin.SetMode(gin.TestMode)
	c := client.New(srv.URL + "/graphql")
	var resp struct {
		User struct{
			gm.Model     `json:",squash"`
			AccountID    gm.UUID  `json:"accountID" gorm:"type:char(12);unique_index:idx_acc_emp;not null"`
			StoreID      *gm.UUID `json:"storeID" gorm:"type:char(12) REFERENCES stores(id) on update no action on delete no action"`
			Empno        string   `json:"empno" gorm:"type:char(12);unique_index:idx_acc_emp;not null"`
			PasswordHash gm.MD5   `json:"-" gorm:"type:char(16);not null"`
			Salt         string   `json:"-" gorm:"type:char(16);not null"`
			RealName     string   `json:"realName" gorm:"type:varchar(50);not null"`
			Role         string   `json:"role" gorm:"type:varchar(50);not null"`
			Store		*model.Store `json:"store"`
			Account     struct{
				ID gm.UUID `json:"id"`
			}
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := c.Post(`query{
					user(id:"5d6e1b8de138231520eb8380"){
						id
						createdAt
  						updatedAt
  						deletedAt
                       accountId
                       empno
                       realName
                       role
					}
				}`,
			&resp, func(r *client.Request) {
				r.Header.Set("ACCESS-TOKEN", "eyJUb2tlblZhbHVlcyI6eyJBZ2VudEhhc2giOiI4NWE0MGYyOTMyMGM0ZDlmIiwiR3JhbnRUeXBlIjoiYWNjZXNzIiwiTWV0YWRhdGEiOnsiRW1wbm8iOiJyb290IiwiUGFzc3dvcmRIYXNoIjoiYjdkYTQ1ZTI5NzVjM2YxMjk3MWUzYjNkMzlhNzI4ODMiLCJSb2xlIjoiYWRtaW4iLCJFeHBpcmVzQXQiOiIxOTcwLTAxLTAyVDA4OjE2OjQwLjAwMDAwMSswODowMCJ9fSwiRXhwaXJlc0F0Ijo0NjAwLCJJc3N1ZWRBdCI6MTAwMH0p1ixNltFkVkIrZvxYtTQZOiDpTA==")
			})
		if err != nil {
			b.Fatal(err)
		}
	}
	b.Log(m.CacheStatus())
}