import React, { createRef, PureComponent } from 'react';
import { Invitee } from 'app/types';

export interface Props {
  invitees: Invitee[];
  onRevokeInvite: (code: string) => void;
}

export default class InviteesTable extends PureComponent<Props> {
  private copyUrlRef = createRef<HTMLTextAreaElement>();

  copyToClipboard = () => {
    const node = this.copyUrlRef.current;

    if (node) {
      node.select();
      document.execCommand('copy');
    }
  };

  render() {
    const { invitees, onRevokeInvite } = this.props;

    return (
      <table className="filter-table form-inline">
        <thead>
          <tr>
            <th>邮箱</th>
            <th>姓名</th>
            <th />
            <th style={{ width: '34px' }} />
          </tr>
        </thead>
        <tbody>
          {invitees.map((invitee, index) => {
            return (
              <tr key={`${invitee.id}-${index}`}>
                <td>{invitee.email}</td>
                <td>{invitee.name}</td>
                <td className="text-right">
                  <button className="btn btn-inverse btn-mini" onClick={this.copyToClipboard}>
                    <textarea
                      readOnly={true}
                      value={invitee.url}
                      style={{ position: 'absolute', right: -1000 }}
                      ref={this.copyUrlRef}
                    />
                    <i className="fa fa-clipboard" /> 复制邀请
                  </button>
                  &nbsp;
                </td>
                <td>
                  <button className="btn btn-danger btn-mini" onClick={() => onRevokeInvite(invitee.code)}>
                    <i className="fa fa-remove" />
                  </button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    );
  }
}
