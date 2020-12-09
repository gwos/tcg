#!/usr/bin/perl
print XORpass( pack "H*", $ARGV[0] );

sub XORpass {
    my $k = 'change for more security';
	my $r = '';
	for my $ch (split //, $_[0]){
		my $i = chop $k;
		$r .= chr(ord($ch) ^ ord($i));
		$k = $i . $k;
	}
	return $r;
}