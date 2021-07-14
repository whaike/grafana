import type { Suggestion, Label } from './suggestions';
import { HistoryItem } from '@grafana/data';
import { PromQuery } from '../../types';
import type PLP from '../../language_provider';
import { NeverCaseError } from './util';
import { FUNCTIONS } from '../../promql';

type PromHistoryItem = HistoryItem<PromQuery>;

type Completion = {
  label: string;
  insertText: string;
  suggestOnInsert: boolean;
};

// FIXME: move this to somewhere else maybe?
export function unwrap<T>(value: T | null | undefined): T {
  if (value == null) {
    throw new Error('value must not be nullish');
  }
  return value;
}

async function getAllMetricNamesCompletions(plp: PLP): Promise<Completion[]> {
  const { metricsMetadata } = plp;
  const texts = metricsMetadata == null ? [] : Object.keys(metricsMetadata);
  return texts.map((text) => ({
    label: text,
    insertText: text,
    suggestOnInsert: false,
  }));
}

function getAllFunctionsCompletions(): Completion[] {
  return FUNCTIONS.map((f) => ({
    label: f.label,
    insertText: unwrap(f.insertText),
    suggestOnInsert: false,
  }));
}

function getAllDurationsCompletions(): Completion[] {
  // FIXME: get a better list
  return ['5m', '1m', '30s', '15s'].map((text) => ({
    label: text,
    insertText: text,
    suggestOnInsert: false,
  }));
}

function getAllHistoryCompletions(queryHistory: PromHistoryItem[]): Completion[] {
  // NOTE: the typescript types are wrong. historyItem.query.expr can be undefined
  const exprs = queryHistory
    .map((h) => h.query.expr)
    .filter((expr) => expr !== undefined)
    .slice(0, 10); // FIXME: better limit

  return exprs.map((expr) => ({
    label: expr,
    insertText: expr,
    suggestOnInsert: false,
  }));
}

function makeSelector(metricName: string, labels: Label[]): string {
  // FIXME: check if this deals well with usually-escaped-non-ascii things
  const labelTexts = labels.map((label) => `${label.name}="${label.value}"`);
  return `{__name__="${metricName}",${labelTexts.join(',')}}`;
}

async function getLabelNamesForCompletions(
  metric: string,
  suffix: string,
  suggestOnInsert: boolean,
  otherLabels: Label[],
  plp: PLP
): Promise<Completion[]> {
  const selector = makeSelector(metric, otherLabels);
  const data = await plp.getSeries(selector);
  const possibleLabelNames = Object.keys(data); // all names from prometheus
  const usedLabelNames = new Set(otherLabels.map((l) => l.name)); // names used in the query
  const labelNames = possibleLabelNames.filter((l) => !usedLabelNames.has(l));
  return labelNames.map((text) => ({
    label: text,
    insertText: `${text}${suffix}`,
    suggestOnInsert,
  }));
}

async function getLabelNamesForSelectorCompletions(
  metric: string,
  otherLabels: Label[],
  plp: PLP
): Promise<Completion[]> {
  return getLabelNamesForCompletions(metric, '=', true, otherLabels, plp);
}
async function getLabelNamesForByCompletions(metric: string, otherLabels: Label[], plp: PLP): Promise<Completion[]> {
  return getLabelNamesForCompletions(metric, '', false, otherLabels, plp);
}

async function getLabelValuesForMetricCompletions(
  metric: string,
  labelName: string,
  otherLabels: Label[],
  plp: PLP
): Promise<Completion[]> {
  const selector = makeSelector(metric, otherLabels);
  const data = await plp.getSeries(selector);
  const values = data[labelName] ?? [];
  return values.map((text) => ({
    label: text,
    insertText: `"${text}"`, // FIXME: escaping strange characters?
    suggestOnInsert: false,
  }));
}

export async function getCompletions(
  suggestion: Suggestion,
  plp: PLP,
  queryHistory: PromHistoryItem[]
): Promise<Completion[]> {
  console.log(`getting completions for ${JSON.stringify(suggestion)}`);
  switch (suggestion.type) {
    case 'ALL_DURATIONS':
      return getAllDurationsCompletions(plp);
    case 'ALL_METRIC_NAMES':
      return getAllMetricNamesCompletions(plp);
    case 'ALL_METRIC_AND_FUNCTION_NAMES': {
      const metricNames = await getAllMetricNamesCompletions(plp);
      return [...metricNames, ...getAllFunctionsCompletions()];
    }
    case 'ALL_METRIC_AND_FUNCTION_NAMES_AND_HISTORY': {
      const metricNames = await getAllMetricNamesCompletions(plp);
      return [...metricNames, ...getAllFunctionsCompletions(), ...getAllHistoryCompletions(queryHistory)];
    }
    case 'LABEL_NAMES_FOR_SELECTOR':
      return getLabelNamesForSelectorCompletions(suggestion.metricName, suggestion.otherLabels, plp);
    case 'LABEL_NAMES_FOR_BY':
      return getLabelNamesForByCompletions(suggestion.metricName, suggestion.otherLabels, plp);
    case 'LABEL_VALUES':
      return getLabelValuesForMetricCompletions(
        suggestion.metricName,
        suggestion.labelName,
        suggestion.otherLabels,
        plp
      );
    default:
      throw new NeverCaseError(suggestion);
  }
}
