#!/usr/bin/perl

use Modern::Perl;

# get args from commandline

say '<html><head><meta http-equiv="refresh" content="30"></head><body><h1>Hastypoloniexbot Report</h1><pre>';

for my $i  (0..$#ARGV) {
    my $file=$ARGV[$i];
    say $file."\n";
    say `./statesummary $file\n`;

    my $dur= `jq ._TOTAL_.Misc $file`;
    my $startbalance=  `jq ._START_.Balance $file`;
    my $startcoin=  `jq ._START_.Coin $file`;
    my $totalbalance= `jq ._TOTAL_.Balance $file`;
    my $totalfiat=  `jq ._TOTAL_.FiatValue $file`;
    my $growth=int((($totalbalance-$startbalance)/$startbalance)*10000)/100;
     chomp $startbalance; chomp $startcoin; chomp $dur;
     #say $dur;
     no warnings;
     my ($hours,$rest,$trim) = $dur =~  m/ " (\d+ h)? ([\w \d]+) (\. \d+ s ) "$ /x;
     $startcoin =~ s/"(\w+)"/$1/;
    # say "hours $hours rest $rest trim $trim";
     my $days=int($hours)/24;
     $hours=$hours % 24;
    say "\nDuration: $days"."d$hours"."h$rest"."s Start Balance: $startbalance $startcoin Growth: $growth%\n";
}

say '</pre></body></html>';
