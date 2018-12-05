import moment from 'moment';
import { PostgresDatasource } from '../datasource';
import { CustomVariable } from 'app/features/templating/custom_variable';

describe('PostgreSQLDatasource', () => {
  const instanceSettings = { name: 'postgresql' };

  const backendSrv = {};
  const templateSrv = {
    replace: jest.fn(text => text),
  };
  const raw = {
    from: moment.utc('2018-04-25 10:00'),
    to: moment.utc('2018-04-25 11:00'),
  };
  const ctx = {
    backendSrv,
    timeSrvMock: {
      timeRange: () => ({
        from: raw.from,
        to: raw.to,
        raw: raw,
      }),
    },
  } as any;

  beforeEach(() => {
    ctx.ds = new PostgresDatasource(instanceSettings, backendSrv, {}, templateSrv, ctx.timeSrvMock);
  });

  describe('When performing annotationQuery', () => {
    let results;

    const annotationName = 'MyAnno';

    const options = {
      annotation: {
        name: annotationName,
        rawQuery: 'select time, title, text, tags from table;',
      },
      range: {
        from: moment(1432288354),
        to: moment(1432288401),
      },
    };

    const response = {
      results: {
        MyAnno: {
          refId: annotationName,
          tables: [
            {
              columns: [{ text: '时间' }, { text: 'text' }, { text: 'tags' }],
              rows: [
                [1432288355, 'some text', 'TagA,TagB'],
                [1432288390, 'some text2', ' TagB , TagC'],
                [1432288400, 'some text3'],
              ],
            },
          ],
        },
      },
    };

    beforeEach(() => {
      ctx.backendSrv.datasourceRequest = jest.fn(options => {
        return Promise.resolve({ data: response, status: 200 });
      });
      ctx.ds.annotationQuery(options).then(data => {
        results = data;
      });
    });

    it('应该返回注释列表', () => {
      expect(results.length).toBe(3);

      expect(results[0].text).toBe('some text');
      expect(results[0].tags[0]).toBe('TagA');
      expect(results[0].tags[1]).toBe('TagB');

      expect(results[1].tags[0]).toBe('TagB');
      expect(results[1].tags[1]).toBe('TagC');

      expect(results[2].tags.length).toBe(0);
    });
  });

  describe('执行metricFindQuery时', () => {
    let results;
    const query = 'select * from atable';
    const response = {
      results: {
        tempvar: {
          meta: {
            rowCount: 3,
          },
          refId: 'tempvar',
          tables: [
            {
              columns: [{ text: 'title' }, { text: 'text' }],
              rows: [['aTitle', 'some text'], ['aTitle2', 'some text2'], ['aTitle3', 'some text3']],
            },
          ],
        },
      },
    };

    beforeEach(() => {
      ctx.backendSrv.datasourceRequest = jest.fn(options => {
        return Promise.resolve({ data: response, status: 200 });
      });
      ctx.ds.metricFindQuery(query).then(data => {
        results = data;
      });
    });

    it('应该返回所有列值的列表', () => {
      expect(results.length).toBe(6);
      expect(results[0].text).toBe('aTitle');
      expect(results[5].text).toBe('some text3');
    });
  });

  describe('使用键，值列执行metricFindQuery时', () => {
    let results;
    const query = 'select * from atable';
    const response = {
      results: {
        tempvar: {
          meta: {
            rowCount: 3,
          },
          refId: 'tempvar',
          tables: [
            {
              columns: [{ text: '__value' }, { text: '__text' }],
              rows: [['value1', 'aTitle'], ['value2', 'aTitle2'], ['value3', 'aTitle3']],
            },
          ],
        },
      },
    };

    beforeEach(() => {
      ctx.backendSrv.datasourceRequest = jest.fn(options => {
        return Promise.resolve({ data: response, status: 200 });
      });
      ctx.ds.metricFindQuery(query).then(data => {
        results = data;
      });
    });

    it('应该返回文本列表，值', () => {
      expect(results.length).toBe(3);
      expect(results[0].text).toBe('aTitle');
      expect(results[0].value).toBe('value1');
      expect(results[2].text).toBe('aTitle3');
      expect(results[2].value).toBe('value3');
    });
  });

  describe('使用键，值列和重复键执行metricFindQuery时', () => {
    let results;
    const query = 'select * from atable';
    const response = {
      results: {
        tempvar: {
          meta: {
            rowCount: 3,
          },
          refId: 'tempvar',
          tables: [
            {
              columns: [{ text: '__text' }, { text: '__value' }],
              rows: [['aTitle', 'same'], ['aTitle', 'same'], ['aTitle', 'diff']],
            },
          ],
        },
      },
    };

    beforeEach(() => {
      ctx.backendSrv.datasourceRequest = jest.fn(options => {
        return Promise.resolve({ data: response, status: 200 });
      });
      ctx.ds.metricFindQuery(query).then(data => {
        results = data;
      });
      //ctx.$rootScope.$apply();
    });

    it('应返回唯一键列表', () => {
      expect(results.length).toBe(1);
      expect(results[0].text).toBe('aTitle');
      expect(results[0].value).toBe('same');
    });
  });

  describe('插值变量时', () => {
    beforeEach(() => {
      ctx.variable = new CustomVariable({}, {});
    });

    describe('值是一个字符串', () => {
      it('应该返回一个不带引号的值', () => {
        expect(ctx.ds.interpolateVariable('abc', ctx.variable)).toEqual('abc');
      });
    });

    describe('值是一个数字', () => {
      it('应该返回一个不带引号的值', () => {
        expect(ctx.ds.interpolateVariable(1000, ctx.variable)).toEqual(1000);
      });
    });

    describe('值是一个字符串数组', () => {
      it('应该返回逗号分隔的引用值', () => {
        expect(ctx.ds.interpolateVariable(['a', 'b', 'c'], ctx.variable)).toEqual("'a','b','c'");
      });
    });

    describe('变量允许多值，是一个字符串', () => {
      it('应该返回一个带引号的值', () => {
        ctx.variable.multi = true;
        expect(ctx.ds.interpolateVariable('abc', ctx.variable)).toEqual("'abc'");
      });
    });

    describe('变量包含单引号', () => {
      it('应该返回一个带引号的值', () => {
        ctx.variable.multi = true;
        expect(ctx.ds.interpolateVariable("a'bc", ctx.variable)).toEqual("'a''bc'");
        expect(ctx.ds.interpolateVariable("a'b'c", ctx.variable)).toEqual("'a''b''c'");
      });
    });

    describe('变量允许all并且是一个字符串', () => {
      it('应该返回一个带引号的值', () => {
        ctx.variable.includeAll = true;
        expect(ctx.ds.interpolateVariable('abc', ctx.variable)).toEqual("'abc'");
      });
    });
  });
});
