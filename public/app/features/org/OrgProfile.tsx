import React, { SFC } from 'react';

export interface Props {
  orgName: string;
  onSubmit: () => void;
  onOrgNameChange: (orgName: string) => void;
}

const OrgProfile: SFC<Props> = ({ onSubmit, onOrgNameChange, orgName }) => {
  return (
    <div>
      <h3 className="page-sub-heading">组织简介</h3>
      <form
        name="orgForm"
        className="gf-form-group"
        onSubmit={event => {
          event.preventDefault();
          onSubmit();
        }}
      >
        <div className="gf-form-inline">
          <div className="gf-form max-width-28">
            <span className="gf-form-label">组织名称</span>
            <input
              className="gf-form-input"
              type="text"
              onChange={event => {
                onOrgNameChange(event.target.value);
              }}
              value={orgName}
            />
          </div>
        </div>
        <div className="gf-form-button-row">
          <button type="submit" className="btn btn-success">
            保存
          </button>
        </div>
      </form>
    </div>
  );
};

export default OrgProfile;
