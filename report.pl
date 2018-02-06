#!/usr/bin/perl

use Modern::Perl;

# get args from commandline

say "<html><head></head><body><h1>Hastypoloniexbot Report</h1><pre>";

for my $i  (0..$#ARGV) {
    my $file=$ARGV[$i];
    say $file."\n";
    say `./statesummary $file\n`;
}

say "</pre></body></html>"
