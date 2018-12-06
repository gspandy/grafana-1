package api

import (
	"github.com/grafana/grafana/pkg/api/dtos"
	"github.com/grafana/grafana/pkg/bus"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/util"
)

// GET /api/teams/:teamId/members
func GetTeamMembers(c *m.ReqContext) Response {
	query := m.GetTeamMembersQuery{OrgId: c.OrgId, TeamId: c.ParamsInt64(":teamId")}

	if err := bus.Dispatch(&query); err != nil {
		return Error(500, "未能获得团队成员", err)
	}

	for _, member := range query.Result {
		member.AvatarUrl = dtos.GetGravatarUrl(member.Email)
		member.Labels = []string{}

		if setting.IsEnterprise && setting.LdapEnabled && member.External {
			member.Labels = append(member.Labels, "LDAP")
		}
	}

	return JSON(200, query.Result)
}

// POST /api/teams/:teamId/members
func AddTeamMember(c *m.ReqContext, cmd m.AddTeamMemberCommand) Response {
	cmd.TeamId = c.ParamsInt64(":teamId")
	cmd.OrgId = c.OrgId

	if err := bus.Dispatch(&cmd); err != nil {
		if err == m.ErrTeamNotFound {
			return Error(404, "团队未找到", nil)
		}

		if err == m.ErrTeamMemberAlreadyAdded {
			return Error(400, "用户已添加到此团队中", nil)
		}

		return Error(500, "无法将用户添加到团队", err)
	}

	return JSON(200, &util.DynMap{
		"message": "用户加入团队成功",
	})
}

// DELETE /api/teams/:teamId/members/:userId
func RemoveTeamMember(c *m.ReqContext) Response {
	if err := bus.Dispatch(&m.RemoveTeamMemberCommand{OrgId: c.OrgId, TeamId: c.ParamsInt64(":teamId"), UserId: c.ParamsInt64(":userId")}); err != nil {
		if err == m.ErrTeamNotFound {
			return Error(404, "团队未找到", nil)
		}

		if err == m.ErrTeamMemberNotFound {
			return Error(404, "未找到团队成员", nil)
		}

		return Error(500, "无法从团队中删除成员", err)
	}
	return Success("成员已删除")
}
