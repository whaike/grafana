import { css } from '@emotion/css';
import { DataSourceApi, ExploreQueryFieldProps, SelectableValue } from '@grafana/data';
import { selectors } from '@grafana/e2e-selectors';
import { config, getDataSourceSrv } from '@grafana/runtime';
import {
  BracesPlugin,
  FileDropzone,
  InlineField,
  InlineFieldRow,
  InlineLabel,
  Input,
  LegacyForms,
  QueryField,
  RadioButtonGroup,
  SlatePrism,
  Themeable2,
  TypeaheadInput,
  TypeaheadOutput,
  withTheme2,
} from '@grafana/ui';
import { TraceToLogsOptions } from 'app/core/components/TraceToLogsSettings';
import Prism from 'prismjs';
import React from 'react';
import { Node } from 'slate';
import { LokiQueryField } from '../loki/components/LokiQueryField';
import { LokiQuery } from '../loki/types';
import { TempoDatasource, TempoQuery, TempoQueryType } from './datasource';
import { tokenizer } from './syntax';

interface Props extends ExploreQueryFieldProps<TempoDatasource, TempoQuery>, Themeable2 {}

const DEFAULT_QUERY_TYPE: TempoQueryType = 'traceId';
interface State {
  linkedDatasource?: DataSourceApi;
  hasSyntaxLoaded: boolean;
}

const PRISM_LANGUAGE = 'tempo';
const durationPlaceholder = 'e.g. 1.2s, 100ms, 500us';
const plugins = [
  BracesPlugin(),
  SlatePrism({
    onlyIn: (node: Node) => node.object === 'block' && node.type === 'code_block',
    getSyntax: () => PRISM_LANGUAGE,
  }),
];

Prism.languages[PRISM_LANGUAGE] = tokenizer;

class TempoQueryFieldComponent extends React.PureComponent<Props, State> {
  state = {
    linkedDatasource: undefined,
    hasSyntaxLoaded: false,
  };

  constructor(props: Props) {
    super(props);
  }

  async componentDidMount() {
    const { datasource } = this.props;
    // Find query field from linked datasource
    const tracesToLogsOptions: TraceToLogsOptions = datasource.tracesToLogs || {};
    const linkedDatasourceUid = tracesToLogsOptions.datasourceUid;
    if (linkedDatasourceUid) {
      const dsSrv = getDataSourceSrv();
      const linkedDatasource = await dsSrv.get(linkedDatasourceUid);
      this.setState({
        linkedDatasource,
      });
    }

    if (config.featureToggles.tempoSearch) {
      await this.fetchAutocomplete();
    }
  }

  async fetchAutocomplete() {
    await this.props.datasource.languageProvider.start();
    this.setState({ hasSyntaxLoaded: true });
  }

  onChangeLinkedQuery = (value: LokiQuery) => {
    const { query, onChange } = this.props;
    onChange({
      ...query,
      linkedQuery: { ...value, refId: 'linked' },
    });
  };

  onRunLinkedQuery = () => {
    this.props.onRunQuery();
  };

  onTypeahead = async (typeahead: TypeaheadInput): Promise<TypeaheadOutput> => {
    const languageProvider = this.props.datasource.languageProvider;
    return await languageProvider.provideCompletionItems(typeahead);
  };

  cleanText = (text: string) => {
    const splittedText = text.split(/\s+(?=([^"]*"[^"]*")*[^"]*$)/g);
    if (splittedText.length > 1) {
      return splittedText[splittedText.length - 1];
    }
    return text;
  };

  render() {
    const { query, onChange } = this.props;
    const { linkedDatasource } = this.state;

    const queryTypeOptions: Array<SelectableValue<TempoQueryType>> = [
      { value: 'search', label: 'Search' },
      { value: 'traceId', label: 'TraceID' },
      { value: 'upload', label: 'JSON file' },
    ];

    return (
      <>
        <InlineFieldRow>
          <InlineField label="Query type">
            <RadioButtonGroup<TempoQueryType>
              options={queryTypeOptions}
              value={query.queryType || DEFAULT_QUERY_TYPE}
              onChange={(v) =>
                onChange({
                  ...query,
                  queryType: v,
                })
              }
              size="md"
            />
          </InlineField>
        </InlineFieldRow>
        {query.queryType === 'search' && (
          <>
            {config.featureToggles.tempoSearch ? (
              <div className={css({ maxWidth: '500px' })}>
                <InlineFieldRow>
                  <InlineField label="Service Name" labelWidth={14} grow>
                    <QueryField
                      additionalPlugins={plugins}
                      query={query.search}
                      onTypeahead={this.onTypeahead}
                      onBlur={this.props.onBlur}
                      onChange={(value) => {
                        onChange({
                          ...query,
                          search: value,
                        });
                      }}
                      cleanText={this.cleanText}
                      onRunQuery={this.onRunLinkedQuery}
                      syntaxLoaded={this.state.hasSyntaxLoaded}
                      portalOrigin="tempo"
                    />
                  </InlineField>
                </InlineFieldRow>
                <InlineFieldRow>
                  <InlineField label="Span Name" labelWidth={14} grow>
                    <QueryField
                      additionalPlugins={plugins}
                      query={query.search}
                      onTypeahead={this.onTypeahead}
                      onBlur={this.props.onBlur}
                      onChange={(value) => {
                        onChange({
                          ...query,
                          search: value,
                        });
                      }}
                      cleanText={this.cleanText}
                      onRunQuery={this.onRunLinkedQuery}
                      syntaxLoaded={this.state.hasSyntaxLoaded}
                      portalOrigin="tempo"
                    />
                  </InlineField>
                </InlineFieldRow>
                <InlineFieldRow>
                  <InlineField label="Tags" labelWidth={14} grow tooltip="Values should be in the logfmt format.">
                    <QueryField
                      additionalPlugins={plugins}
                      query={query.search}
                      onTypeahead={this.onTypeahead}
                      onBlur={this.props.onBlur}
                      onChange={(value) => {
                        onChange({
                          ...query,
                          search: value,
                        });
                      }}
                      cleanText={this.cleanText}
                      onRunQuery={this.onRunLinkedQuery}
                      syntaxLoaded={this.state.hasSyntaxLoaded}
                      portalOrigin="tempo"
                    />
                  </InlineField>
                </InlineFieldRow>
                <InlineFieldRow>
                  <InlineField label="Min Duration" labelWidth={14} grow>
                    <Input
                      value={query.minDuration || ''}
                      placeholder={durationPlaceholder}
                      onChange={(v) =>
                        onChange({
                          ...query,
                          minDuration: v.currentTarget.value,
                        })
                      }
                    />
                  </InlineField>
                </InlineFieldRow>
                <InlineFieldRow>
                  <InlineField label="Max Duration" labelWidth={14} grow>
                    <Input
                      value={query.maxDuration || ''}
                      placeholder={durationPlaceholder}
                      onChange={(v) =>
                        onChange({
                          ...query,
                          maxDuration: v.currentTarget.value,
                        })
                      }
                    />
                  </InlineField>
                </InlineFieldRow>
                <InlineFieldRow>
                  <InlineField label="Limit" labelWidth={14} grow tooltip="Maximum numbers of returned results">
                    <Input
                      value={query.limit || ''}
                      type="number"
                      onChange={(v) =>
                        onChange({
                          ...query,
                          limit: v.currentTarget.value ? parseInt(v.currentTarget.value, 10) : undefined,
                        })
                      }
                    />
                  </InlineField>
                </InlineFieldRow>
              </div>
            ) : (
              <>
                {!linkedDatasource ? (
                  <div className="text-warning">
                    Please set up a Traces-to-logs datasource in the datasource settings.
                  </div>
                ) : (
                  <>
                    <InlineLabel>
                      Tempo uses {((linkedDatasource as unknown) as DataSourceApi).name} to find traces.
                    </InlineLabel>

                    <LokiQueryField
                      datasource={linkedDatasource!}
                      onChange={this.onChangeLinkedQuery}
                      onRunQuery={this.onRunLinkedQuery}
                      query={this.props.query.linkedQuery ?? ({ refId: 'linked' } as any)}
                      history={[]}
                    />
                  </>
                )}
              </>
            )}
          </>
        )}

        {query.queryType === 'upload' && (
          <div className={css({ padding: this.props.theme.spacing(2) })}>
            <FileDropzone
              options={{ multiple: false }}
              onLoad={(result) => {
                this.props.datasource.uploadedJson = result;
                this.props.onRunQuery();
              }}
            />
          </div>
        )}
        {(!query.queryType || query.queryType === 'traceId') && (
          <LegacyForms.FormField
            label="Trace ID"
            labelWidth={4}
            inputEl={
              <div className="slate-query-field__wrapper">
                <div className="slate-query-field" aria-label={selectors.components.QueryField.container}>
                  <input
                    style={{ width: '100%' }}
                    value={query.query || ''}
                    onChange={(e) =>
                      onChange({
                        ...query,
                        query: e.currentTarget.value,
                        queryType: 'traceId',
                        linkedQuery: undefined,
                      })
                    }
                  />
                </div>
              </div>
            }
          />
        )}
      </>
    );
  }
}

export const TempoQueryField = withTheme2(TempoQueryFieldComponent);
