export const getMockNavModel = (pageName: string) => {
  return {
    node: {
      active: false,
      icon: 'gicon gicon-team',
      id: `team-${pageName}-2`,
      text: `${pageName}`,
      url: 'org/teams/edit/2/members',
      parentItem: {
        img: '/avatar/b5695b61c91d13e7fa2fe71cfb95de9b',
        id: 'team-2',
        subTitle: '管理成员 & 设置',
        url: '',
        text: 'test1',
        breadcrumbs: [{ title: '团队', url: 'org/teams' }],
        children: [
          {
            active: false,
            icon: 'gicon gicon-team',
            id: 'team-members-2',
            text: '成员',
            url: 'org/teams/edit/2/members',
          },
          {
            active: false,
            icon: 'fa fa-fw fa-sliders',
            id: 'team-settings-2',
            text: '设置',
            url: 'org/teams/edit/2/settings',
          },
        ],
      },
    },
    main: {
      img: '/avatar/b5695b61c91d13e7fa2fe71cfb95de9b',
      id: 'team-2',
      subTitle: '管理成员和设置',
      url: '',
      text: 'test1',
      breadcrumbs: [{ title: '团队', url: 'org/teams' }],
      children: [
        {
          active: true,
          icon: 'gicon gicon-team',
          id: 'team-members-2',
          text: '成员',
          url: 'org/teams/edit/2/members',
        },
        {
          active: false,
          icon: 'fa fa-fw fa-sliders',
          id: 'team-settings-2',
          text: '设置',
          url: 'org/teams/edit/2/settings',
        },
      ],
    },
  };
};
