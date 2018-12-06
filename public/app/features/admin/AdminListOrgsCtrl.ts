
export default class AdminListOrgsCtrl {
  /** @ngInject */
  constructor($scope, backendSrv, navModelSrv) {
    $scope.init = () => {
      $scope.navModel = navModelSrv.getNav('cfg', 'admin', 'global-orgs', 1);
      $scope.getOrgs();
    };

    $scope.getOrgs = () => {
      backendSrv.get('/api/orgs').then(orgs => {
        $scope.orgs = orgs;
      });
    };

    $scope.deleteOrg = org => {
      $scope.appEvent('confirm-modal', {
        title: '删除',
        text: '是否确定删除组织 ' + org.name + '?',
        text2: '该组织的所有仪表板将被删除！',
        icon: 'fa-trash',
        yesText: '删除',
        onConfirm: () => {
          backendSrv.delete('/api/orgs/' + org.id).then(() => {
            $scope.getOrgs();
          });
        },
      });
    };

    $scope.init();
  }
}

