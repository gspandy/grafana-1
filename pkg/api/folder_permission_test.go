package api

import (
	"testing"

	"github.com/grafana/grafana/pkg/api/dtos"
	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/components/simplejson"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/guardian"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFolderPermissionApiEndpoint(t *testing.T) {
	Convey("文件夹权限测试", t, func() {
		Convey("给定文件夹不存在", func() {
			mock := &fakeFolderService{
				GetFolderByUIDError: m.ErrFolderNotFound,
			}

			origNewFolderService := dashboards.NewFolderService
			mockFolderService(mock)

			loggedInUserScenarioWithRole("当发生GET请求时", "GET", "/api/folders/uid/permissions", "/api/folders/:uid/permissions", m.ROLE_EDITOR, func(sc *scenarioContext) {
				callGetFolderPermissions(sc)
				So(sc.resp.Code, ShouldEqual, 404)
			})

			cmd := dtos.UpdateDashboardAclCommand{
				Items: []dtos.DashboardAclUpdateItem{
					{UserId: 1000, Permission: m.PERMISSION_ADMIN},
				},
			}

			updateFolderPermissionScenario("当发生POST请求时", "/api/folders/uid/permissions", "/api/folders/:uid/permissions", cmd, func(sc *scenarioContext) {
				callUpdateFolderPermissions(sc)
				So(sc.resp.Code, ShouldEqual, 404)
			})

			Reset(func() {
				dashboards.NewFolderService = origNewFolderService
			})
		})

		Convey("给定用户没有管理员权限", func() {
			origNewGuardian := guardian.New
			guardian.MockDashboardGuardian(&guardian.FakeDashboardGuardian{CanAdminValue: false})

			mock := &fakeFolderService{
				GetFolderByUIDResult: &m.Folder{
					Id:    1,
					Uid:   "uid",
					Title: "文件夹",
				},
			}

			origNewFolderService := dashboards.NewFolderService
			mockFolderService(mock)

			loggedInUserScenarioWithRole("当发生GET请求时", "GET", "/api/folders/uid/permissions", "/api/folders/:uid/permissions", m.ROLE_EDITOR, func(sc *scenarioContext) {
				callGetFolderPermissions(sc)
				So(sc.resp.Code, ShouldEqual, 403)
			})

			cmd := dtos.UpdateDashboardAclCommand{
				Items: []dtos.DashboardAclUpdateItem{
					{UserId: 1000, Permission: m.PERMISSION_ADMIN},
				},
			}

			updateFolderPermissionScenario("当发生POST请求时", "/api/folders/uid/permissions", "/api/folders/:uid/permissions", cmd, func(sc *scenarioContext) {
				callUpdateFolderPermissions(sc)
				So(sc.resp.Code, ShouldEqual, 403)
			})

			Reset(func() {
				guardian.New = origNewGuardian
				dashboards.NewFolderService = origNewFolderService
			})
		})

		Convey("给定用户具有管理员权限和更新权限", func() {
			origNewGuardian := guardian.New
			guardian.MockDashboardGuardian(&guardian.FakeDashboardGuardian{
				CanAdminValue:                    true,
				CheckPermissionBeforeUpdateValue: true,
				GetAclValue: []*m.DashboardAclInfoDTO{
					{OrgId: 1, DashboardId: 1, UserId: 2, Permission: m.PERMISSION_VIEW},
					{OrgId: 1, DashboardId: 1, UserId: 3, Permission: m.PERMISSION_EDIT},
					{OrgId: 1, DashboardId: 1, UserId: 4, Permission: m.PERMISSION_ADMIN},
					{OrgId: 1, DashboardId: 1, TeamId: 1, Permission: m.PERMISSION_VIEW},
					{OrgId: 1, DashboardId: 1, TeamId: 2, Permission: m.PERMISSION_ADMIN},
				},
			})

			mock := &fakeFolderService{
				GetFolderByUIDResult: &m.Folder{
					Id:    1,
					Uid:   "uid",
					Title: "文件夹",
				},
			}

			origNewFolderService := dashboards.NewFolderService
			mockFolderService(mock)

			loggedInUserScenarioWithRole("当发生GET请求时", "GET", "/api/folders/uid/permissions", "/api/folders/:uid/permissions", m.ROLE_ADMIN, func(sc *scenarioContext) {
				callGetFolderPermissions(sc)
				So(sc.resp.Code, ShouldEqual, 200)
				respJSON, err := simplejson.NewJson(sc.resp.Body.Bytes())
				So(err, ShouldBeNil)
				So(len(respJSON.MustArray()), ShouldEqual, 5)
				So(respJSON.GetIndex(0).Get("userId").MustInt(), ShouldEqual, 2)
				So(respJSON.GetIndex(0).Get("permission").MustInt(), ShouldEqual, m.PERMISSION_VIEW)
			})

			cmd := dtos.UpdateDashboardAclCommand{
				Items: []dtos.DashboardAclUpdateItem{
					{UserId: 1000, Permission: m.PERMISSION_ADMIN},
				},
			}

			updateFolderPermissionScenario("当发生POST请求时", "/api/folders/uid/permissions", "/api/folders/:uid/permissions", cmd, func(sc *scenarioContext) {
				callUpdateFolderPermissions(sc)
				So(sc.resp.Code, ShouldEqual, 200)
			})

			Reset(func() {
				guardian.New = origNewGuardian
				dashboards.NewFolderService = origNewFolderService
			})
		})

		Convey("尝试使用重复权限更新权限时", func() {
			origNewGuardian := guardian.New
			guardian.MockDashboardGuardian(&guardian.FakeDashboardGuardian{
				CanAdminValue:                    true,
				CheckPermissionBeforeUpdateValue: false,
				CheckPermissionBeforeUpdateError: guardian.ErrGuardianPermissionExists,
			})

			mock := &fakeFolderService{
				GetFolderByUIDResult: &m.Folder{
					Id:    1,
					Uid:   "uid",
					Title: "文件夹",
				},
			}

			origNewFolderService := dashboards.NewFolderService
			mockFolderService(mock)

			cmd := dtos.UpdateDashboardAclCommand{
				Items: []dtos.DashboardAclUpdateItem{
					{UserId: 1000, Permission: m.PERMISSION_ADMIN},
				},
			}

			updateFolderPermissionScenario("当发生POST请求时", "/api/folders/uid/permissions", "/api/folders/:uid/permissions", cmd, func(sc *scenarioContext) {
				callUpdateFolderPermissions(sc)
				So(sc.resp.Code, ShouldEqual, 400)
			})

			Reset(func() {
				guardian.New = origNewGuardian
				dashboards.NewFolderService = origNewFolderService
			})
		})

		Convey("尝试使用较低的presedence覆盖继承的权限时", func() {
			origNewGuardian := guardian.New
			guardian.MockDashboardGuardian(&guardian.FakeDashboardGuardian{
				CanAdminValue:                    true,
				CheckPermissionBeforeUpdateValue: false,
				CheckPermissionBeforeUpdateError: guardian.ErrGuardianOverride},
			)

			mock := &fakeFolderService{
				GetFolderByUIDResult: &m.Folder{
					Id:    1,
					Uid:   "uid",
					Title: "文件夹",
				},
			}

			origNewFolderService := dashboards.NewFolderService
			mockFolderService(mock)

			cmd := dtos.UpdateDashboardAclCommand{
				Items: []dtos.DashboardAclUpdateItem{
					{UserId: 1000, Permission: m.PERMISSION_ADMIN},
				},
			}

			updateFolderPermissionScenario("当发生POST请求时", "/api/folders/uid/permissions", "/api/folders/:uid/permissions", cmd, func(sc *scenarioContext) {
				callUpdateFolderPermissions(sc)
				So(sc.resp.Code, ShouldEqual, 400)
			})

			Reset(func() {
				guardian.New = origNewGuardian
				dashboards.NewFolderService = origNewFolderService
			})
		})
	})
}

func callGetFolderPermissions(sc *scenarioContext) {
	sc.handlerFunc = GetFolderPermissionList
	sc.fakeReqWithParams("GET", sc.url, map[string]string{}).exec()
}

func callUpdateFolderPermissions(sc *scenarioContext) {
	bus.AddHandler("test", func(cmd *m.UpdateDashboardAclCommand) error {
		return nil
	})

	sc.fakeReqWithParams("POST", sc.url, map[string]string{}).exec()
}

func updateFolderPermissionScenario(desc string, url string, routePattern string, cmd dtos.UpdateDashboardAclCommand, fn scenarioFunc) {
	Convey(desc+" "+url, func() {
		defer bus.ClearBusHandlers()

		sc := setupScenarioContext(url)

		sc.defaultHandler = Wrap(func(c *m.ReqContext) Response {
			sc.context = c
			sc.context.OrgId = TestOrgID
			sc.context.UserId = TestUserID

			return UpdateFolderPermissions(c, cmd)
		})

		sc.m.Post(routePattern, sc.defaultHandler)

		fn(sc)
	})
}
