#!/usr/bin/perl

use Modern::Perl;

# get args from commandline
no warnings;
say '<html><head><meta http-equiv="refresh" content="30"></head><body><h1>Hastypoloniexbot Report</h1><pre>';

for my $i  (0..$#ARGV) {
    my $file=$ARGV[$i];
    say $file."\n";
    my $sum= `./statesummary $file\n`;
    $sum =~ s/ (\d+ h \d+ m \d+ \. \d* s) $/betterdur($1) if $1 ne ""/egmx;
    say $sum;
    
    my $dur= `jq ._TOTAL_.Misc $file`;
    my $startbalance=  `jq ._START_.Balance $file`;
    my $startcoin=  `jq ._START_.Coin $file`;
    my $totalbalance= `jq ._TOTAL_.Balance $file`;
    my $totalfiat=  `jq ._TOTAL_.FiatValue $file`;
    my $growth=int((($totalbalance-$startbalance)/$startbalance)*10000)/100;
    
    chomp $startbalance; chomp $startcoin; chomp $dur;

    my $betterdur=betterdur($dur);
    $startcoin =~ s/"(\w+)"/$1/;
    say "\nGrowth: $growth%  Duration: $betterdur  Start balance: $startbalance $startcoin \n";
}

say '</pre></body></html>';

sub betterdur {
    my $dur=shift ;
    my ($hours,$rest,$trim) = $dur =~  m/ "? (\d+)? h ([\w \d]+) (\. \d+ s ) "?$ /x;
    #say "dur $dur hours $hours rest $rest trim $trim";
    my $days=int($hours/24);
    $hours=$hours % 24;
    return $days."d".$hours."h".$rest."s";
}
