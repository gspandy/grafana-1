import React, { PureComponent } from 'react';
import { connect } from 'react-redux';
import { DataSource, Plugin } from 'app/types';

export interface Props {
  dataSource: DataSource;
  dataSourceMeta: Plugin;
}
interface State {
  name: string;
}

enum DataSourceStates {
  Alpha = 'alpha',
  Beta = 'beta',
}

export class DataSourceSettings extends PureComponent<Props, State> {
  constructor(props) {
    super(props);

    this.state = {
      name: props.dataSource.name,
    };
  }

  onNameChange = event => {
    this.setState({
      name: event.target.value,
    });
  };

  onSubmit = event => {
    event.preventDefault();
    console.log(event);
  };

  onDelete = event => {
    console.log(event);
  };

  isReadyOnly() {
    return this.props.dataSource.readOnly === true;
  }

  shouldRenderInfoBox() {
    const { state } = this.props.dataSourceMeta;

    return state === DataSourceStates.Alpha || state === DataSourceStates.Beta;
  }

  getInfoText() {
    const { dataSourceMeta } = this.props;

    switch (dataSourceMeta.state) {
      case DataSourceStates.Alpha:
        return (
        '此插件被标记为处于alpha状态，这意味着它处于早期开发阶段并更新将包括改变。'
        );

      case DataSourceStates.Beta:
        return (
          '此插件标记为处于beta开发状态。这意味着它目前正在积极开发中，可能缺少重要功能。'
        );
    }

    return null;
  }

  render() {
    const { name } = this.state;

    return (
      <div>
        <h3 className="page-sub-heading">Settings</h3>
        <form onSubmit={this.onSubmit}>
          <div className="gf-form-group">
            <div className="gf-form-inline">
              <div className="gf-form max-width-30">
                <span className="gf-form-label width-10">Name</span>
                <input
                  className="gf-form-input max-width-23"
                  type="text"
                  value={name}
                  placeholder="name"
                  onChange={this.onNameChange}
                  required
                />
              </div>
            </div>
          </div>
          {this.shouldRenderInfoBox() && <div className="grafana-info-box">{this.getInfoText()}</div>}
          {this.isReadyOnly() && (
            <div className="grafana-info-box span8">
              此数据源是由config添加的，无法使用UI进行修改。请联系您的服务器管理员更新此数据源。
            </div>
          )}
          <div className="gf-form-button-row">
            <button type="submit" className="btn btn-success" disabled={this.isReadyOnly()} onClick={this.onSubmit}>
              保存 &amp; 测试
            </button>
            <button type="submit" className="btn btn-danger" disabled={this.isReadyOnly()} onClick={this.onDelete}>
              删除
            </button>
            <a className="btn btn-inverse" href="datasources">
              回退
            </a>
          </div>
        </form>
      </div>
    );
  }
}

function mapStateToProps(state) {
  return {
    dataSource: state.dataSources.dataSource,
    dataSourceMeta: state.dataSources.dataSourceMeta,
  };
}

export default connect(mapStateToProps)(DataSourceSettings);
