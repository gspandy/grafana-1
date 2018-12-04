package sqlstore

import (
	"testing"
	"time"

	"github.com/grafana/grafana/pkg/components/simplejson"
	m "github.com/grafana/grafana/pkg/models"
	. "github.com/smartystreets/goconvey/convey"
)

func mockTimeNow() {
	var timeSeed int64
	timeNow = func() time.Time {
		fakeNow := time.Unix(timeSeed, 0)
		timeSeed++
		return fakeNow
	}
}

func resetTimeNow() {
	timeNow = time.Now
}

func TestAlertingDataAccess(t *testing.T) {
	mockTimeNow()
	defer resetTimeNow()

	Convey("测试告警数据访问", t, func() {
		InitTestDB(t)

		testDash := insertTestDashboard("带告警的仪表板", 1, 0, false, "alert")
		evalData, _ := simplejson.NewJson([]byte(`{"test": "test"}`))
		items := []*m.Alert{
			{
				PanelId:     1,
				DashboardId: testDash.Id,
				OrgId:       testDash.OrgId,
				Name:        "告警标题",
				Message:     "告警信息",
				Settings:    simplejson.New(),
				Frequency:   1,
				EvalData:    evalData,
			},
		}

		cmd := m.SaveAlertsCommand{
			Alerts:      items,
			DashboardId: testDash.Id,
			OrgId:       1,
			UserId:      1,
		}

		err := SaveAlerts(&cmd)

		Convey("可以创建一个告警", func() {
			So(err, ShouldBeNil)
		})

		Convey("可以设置新的状态", func() {
			Convey("新的状态 ok", func() {
				cmd := &m.SetAlertStateCommand{
					AlertId: 1,
					State:   m.AlertStateOK,
				}

				err = SetAlertState(cmd)
				So(err, ShouldBeNil)
			})

			alert, _ := getAlertById(1)
			stateDateBeforePause := alert.NewStateDate

			Convey("可以暂停所有提醒", func() {
				pauseAllAlerts(true)

				Convey("无法更新暂停的提醒", func() {
					cmd := &m.SetAlertStateCommand{
						AlertId: 1,
						State:   m.AlertStateOK,
					}

					err = SetAlertState(cmd)
					So(err, ShouldNotBeNil)
				})

				Convey("暂停告警应更新其NewStateDate", func() {
					alert, _ = getAlertById(1)
					stateDateAfterPause := alert.NewStateDate
					So(stateDateBeforePause, ShouldHappenBefore, stateDateAfterPause)
				})

				Convey("未暂停的告警应该再次更新其NewStateDate", func() {
					pauseAllAlerts(false)
					alert, _ = getAlertById(1)
					stateDateAfterUnpause := alert.NewStateDate
					So(stateDateBeforePause, ShouldHappenBefore, stateDateAfterUnpause)
				})
			})
		})

		Convey("可以读取属性", func() {
			alertQuery := m.GetAlertsQuery{DashboardIDs: []int64{testDash.Id}, PanelId: 1, OrgId: 1, User: &m.SignedInUser{OrgRole: m.ROLE_ADMIN}}
			err2 := HandleAlertsQuery(&alertQuery)

			alert := alertQuery.Result[0]
			So(err2, ShouldBeNil)
			So(alert.Id, ShouldBeGreaterThan, 0)
			So(alert.DashboardId, ShouldEqual, testDash.Id)
			So(alert.PanelId, ShouldEqual, 1)
			So(alert.Name, ShouldEqual, "告警标题")
			So(alert.State, ShouldEqual, m.AlertStateUnknown)
			So(alert.NewStateDate, ShouldNotBeNil)
			So(alert.EvalData, ShouldNotBeNil)
			So(alert.EvalData.Get("test").MustString(), ShouldEqual, "test")
			So(alert.EvalDate, ShouldNotBeNil)
			So(alert.ExecutionError, ShouldEqual, "")
			So(alert.DashboardUid, ShouldNotBeNil)
			So(alert.DashboardSlug, ShouldEqual, "dashboard-with-alerts")
		})

		Convey("Viewer cannot read alerts", func() {
			viewerUser := &m.SignedInUser{OrgRole: m.ROLE_VIEWER, OrgId: 1}
			alertQuery := m.GetAlertsQuery{DashboardIDs: []int64{testDash.Id}, PanelId: 1, OrgId: 1, User: viewerUser}
			err2 := HandleAlertsQuery(&alertQuery)

			So(err2, ShouldBeNil)
			So(alertQuery.Result, ShouldHaveLength, 1)
		})

		Convey("Alerts with same dashboard id and panel id should update", func() {
			modifiedItems := items
			modifiedItems[0].Name = "Name"

			modifiedCmd := m.SaveAlertsCommand{
				DashboardId: testDash.Id,
				OrgId:       1,
				UserId:      1,
				Alerts:      modifiedItems,
			}

			err := SaveAlerts(&modifiedCmd)

			Convey("可以使用相同的仪表板和面板ID保存告警", func() {
				So(err, ShouldBeNil)
			})

			Convey("告警应该更新", func() {
				query := m.GetAlertsQuery{DashboardIDs: []int64{testDash.Id}, OrgId: 1, User: &m.SignedInUser{OrgRole: m.ROLE_ADMIN}}
				err2 := HandleAlertsQuery(&query)

				So(err2, ShouldBeNil)
				So(len(query.Result), ShouldEqual, 1)
				So(query.Result[0].Name, ShouldEqual, "Name")

				Convey("告警状态不应更新", func() {
					So(query.Result[0].State, ShouldEqual, m.AlertStateUnknown)
				})
			})

			Convey("应忽略不更改的更新", func() {
				err3 := SaveAlerts(&modifiedCmd)
				So(err3, ShouldBeNil)
			})
		})

		Convey("每个仪表板有多个告警", func() {
			multipleItems := []*m.Alert{
				{
					DashboardId: testDash.Id,
					PanelId:     1,
					Name:        "1",
					OrgId:       1,
					Settings:    simplejson.New(),
				},
				{
					DashboardId: testDash.Id,
					PanelId:     2,
					Name:        "2",
					OrgId:       1,
					Settings:    simplejson.New(),
				},
				{
					DashboardId: testDash.Id,
					PanelId:     3,
					Name:        "3",
					OrgId:       1,
					Settings:    simplejson.New(),
				},
			}

			cmd.Alerts = multipleItems
			err = SaveAlerts(&cmd)

			Convey("应该保存3个仪表板", func() {
				So(err, ShouldBeNil)

				queryForDashboard := m.GetAlertsQuery{DashboardIDs: []int64{testDash.Id}, OrgId: 1, User: &m.SignedInUser{OrgRole: m.ROLE_ADMIN}}
				err2 := HandleAlertsQuery(&queryForDashboard)

				So(err2, ShouldBeNil)
				So(len(queryForDashboard.Result), ShouldEqual, 3)
			})

			Convey("应该更新两个仪表板并删除一个", func() {
				missingOneAlert := multipleItems[:2]

				cmd.Alerts = missingOneAlert
				err = SaveAlerts(&cmd)

				Convey("应该删除丢失的告警", func() {
					query := m.GetAlertsQuery{DashboardIDs: []int64{testDash.Id}, OrgId: 1, User: &m.SignedInUser{OrgRole: m.ROLE_ADMIN}}
					err2 := HandleAlertsQuery(&query)
					So(err2, ShouldBeNil)
					So(len(query.Result), ShouldEqual, 2)
				})
			})
		})

		Convey("删除仪表板时", func() {
			items := []*m.Alert{
				{
					PanelId:     1,
					DashboardId: testDash.Id,
					Name:        "告警标题",
					Message:     "告警信息",
				},
			}

			cmd := m.SaveAlertsCommand{
				Alerts:      items,
				DashboardId: testDash.Id,
				OrgId:       1,
				UserId:      1,
			}

			SaveAlerts(&cmd)

			err = DeleteDashboard(&m.DeleteDashboardCommand{
				OrgId: 1,
				Id:    testDash.Id,
			})

			So(err, ShouldBeNil)

			Convey("Alerts should be removed", func() {
				query := m.GetAlertsQuery{DashboardIDs: []int64{testDash.Id}, OrgId: 1, User: &m.SignedInUser{OrgRole: m.ROLE_ADMIN}}
				err2 := HandleAlertsQuery(&query)

				So(testDash.Id, ShouldEqual, 1)
				So(err2, ShouldBeNil)
				So(len(query.Result), ShouldEqual, 0)
			})
		})
	})
}

func TestPausingAlerts(t *testing.T) {
	mockTimeNow()
	defer resetTimeNow()

	Convey("给出告警", t, func() {
		InitTestDB(t)

		testDash := insertTestDashboard("带告警的仪表板", 1, 0, false, "alert")
		alert, _ := insertTestAlert("告警标题", "告警信息", testDash.OrgId, testDash.Id, simplejson.New())

		stateDateBeforePause := alert.NewStateDate
		stateDateAfterPause := stateDateBeforePause
		Convey("暂停时", func() {
			pauseAlert(testDash.OrgId, 1, true)

			Convey("应该更新NewStateDate", func() {
				alert, _ := getAlertById(1)

				stateDateAfterPause = alert.NewStateDate
				So(stateDateBeforePause, ShouldHappenBefore, stateDateAfterPause)
			})
		})

		Convey("恢复时", func() {
			pauseAlert(testDash.OrgId, 1, false)

			Convey("应该再次更新NewStateDate", func() {
				alert, _ := getAlertById(1)

				stateDateAfterUnpause := alert.NewStateDate
				So(stateDateAfterPause, ShouldHappenBefore, stateDateAfterUnpause)
			})
		})
	})
}
func pauseAlert(orgId int64, alertId int64, pauseState bool) (int64, error) {
	cmd := &m.PauseAlertCommand{
		OrgId:    orgId,
		AlertIds: []int64{alertId},
		Paused:   pauseState,
	}
	err := PauseAlert(cmd)
	So(err, ShouldBeNil)
	return cmd.ResultCount, err
}
func insertTestAlert(title string, message string, orgId int64, dashId int64, settings *simplejson.Json) (*m.Alert, error) {
	items := []*m.Alert{
		{
			PanelId:     1,
			DashboardId: dashId,
			OrgId:       orgId,
			Name:        title,
			Message:     message,
			Settings:    settings,
			Frequency:   1,
		},
	}

	cmd := m.SaveAlertsCommand{
		Alerts:      items,
		DashboardId: dashId,
		OrgId:       orgId,
		UserId:      1,
	}

	err := SaveAlerts(&cmd)
	return cmd.Alerts[0], err
}

func getAlertById(id int64) (*m.Alert, error) {
	q := &m.GetAlertByIdQuery{
		Id: id,
	}
	err := GetAlertById(q)
	So(err, ShouldBeNil)
	return q.Result, err
}

func pauseAllAlerts(pauseState bool) error {
	cmd := &m.PauseAllAlertCommand{
		Paused: pauseState,
	}
	err := PauseAllAlerts(cmd)
	So(err, ShouldBeNil)
	return err
}
