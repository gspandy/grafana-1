package api

import (
	"github.com/grafana/grafana/pkg/api/dtos"
	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/metrics"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/util"
)

func AdminCreateUser(c *m.ReqContext, form dtos.AdminCreateUserForm) {
	cmd := m.CreateUserCommand{
		Login:    form.Login,
		Email:    form.Email,
		Password: form.Password,
		Name:     form.Name,
	}

	if len(cmd.Login) == 0 {
		cmd.Login = cmd.Email
		if len(cmd.Login) == 0 {
			c.JsonApiErr(400, "验证错误，需要指定用户名或电子邮件", nil)
			return
		}
	}

	if len(cmd.Password) < 4 {
		c.JsonApiErr(400, "密码丢失或太短", nil)
		return
	}

	if err := bus.Dispatch(&cmd); err != nil {
		c.JsonApiErr(500, "无法创建用户", err)
		return
	}

	metrics.M_Api_Admin_User_Create.Inc()

	user := cmd.Result

	result := m.UserIdDTO{
		Message: "创建用户成功",
		Id:      user.Id,
	}

	c.JSON(200, result)
}

func AdminUpdateUserPassword(c *m.ReqContext, form dtos.AdminUpdateUserPasswordForm) {
	userID := c.ParamsInt64(":id")

	if len(form.Password) < 4 {
		c.JsonApiErr(400, "新密码太短", nil)
		return
	}

	userQuery := m.GetUserByIdQuery{Id: userID}

	if err := bus.Dispatch(&userQuery); err != nil {
		c.JsonApiErr(500, "无法从数据库中读取用户", err)
		return
	}

	passwordHashed := util.EncodePassword(form.Password, userQuery.Result.Salt)

	cmd := m.ChangeUserPasswordCommand{
		UserId:      userID,
		NewPassword: passwordHashed,
	}

	if err := bus.Dispatch(&cmd); err != nil {
		c.JsonApiErr(500, "无法更新用户密码", err)
		return
	}

	c.JsonOK("用户密码已更新")
}

func AdminUpdateUserPermissions(c *m.ReqContext, form dtos.AdminUpdateUserPermissionsForm) {
	userID := c.ParamsInt64(":id")

	cmd := m.UpdateUserPermissionsCommand{
		UserId:         userID,
		IsGrafanaAdmin: form.IsGrafanaAdmin,
	}

	if err := bus.Dispatch(&cmd); err != nil {
		c.JsonApiErr(500, "无法更新用户权限", err)
		return
	}

	c.JsonOK("用户权限已更新")
}

func AdminDeleteUser(c *m.ReqContext) {
	userID := c.ParamsInt64(":id")

	cmd := m.DeleteUserCommand{UserId: userID}

	if err := bus.Dispatch(&cmd); err != nil {
		c.JsonApiErr(500, "删除用户失败", err)
		return
	}

	c.JsonOK("用户已删除")
}
