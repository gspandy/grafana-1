import { getBackendSrv } from 'app/core/services/backend_srv';

export interface ServerStat {
  name: string;
  value: number;
}

export const getServerStats = async (): Promise<ServerStat[]> => {
  try {
    const res = await getBackendSrv().get('api/admin/stats');
    return [
      { name: '用户数量', value: res.users },
      { name: '仪表盘数量', value: res.dashboards },
      { name: '活跃用户（最近30天）', value: res.activeUsers },
      { name: '组织数量', value: res.orgs },
      { name: '播放列表数量', value: res.playlists },
      { name: '快照数量', value: res.snapshots },
      { name: '仪表盘标签数量', value: res.tags },
      { name: '标星仪表盘数量', value: res.stars },
      { name: '告警数量', value: res.alerts },
    ];
  } catch (error) {
    console.error(error);
    throw error;
  }
};
