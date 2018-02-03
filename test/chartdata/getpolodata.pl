#!/bin/perl

use Modern::Perl;

my $pair=$ARGV[0] // "BTC_GAS";
my $now=`date +%s`;
my $period=300;
my $periods=100;
my $start=$now-$period*$periods;
my $file=$pair.".json";
say "fetching data for $pair over $periods of $period seconds. Saving to $file";

my $cmd="curl 'https://poloniex.com/public?command=returnChartData&currencyPair=$pair&start=$start&end=9999999999&period=$period' -o $file";
say "Calling: $cmd";

say `$cmd`;
