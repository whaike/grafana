import { MatcherOperatorOptions } from 'app/plugins/datasource/alertmanager/types';
import { TimeZone } from '@grafana/data';

type MatcherField = {
  name: string;
  value: string;
  operator: MatcherOperatorOptions;
};

export type SilenceFormFields = {
  id: string;
  startsAt: string;
  endsAt: string;
  timeZone: TimeZone;
  duration: string;
  comment: string;
  matchers: MatcherField[];
  createdBy: string;
  matcherName: string;
  matcherValue: string;
  isRegex: boolean;
};
