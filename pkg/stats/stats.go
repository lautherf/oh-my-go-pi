package stats

import (
	"regexp"
	"strings"
	"unicode"
)

const rsquo = "\u2019" // right single quotation mark

type UserMessageMetrics struct {
	Chars      int
	Words      int
	Yelling    int
	Profanity  int
	Anguish    int
	Negation   int
	Repetition int
	Blame      int
}

type MessageStats struct {
	ID           int
	SessionFile  string
	EntryID      string
	Folder       string
	Model        string
	Provider     string
	API          string
	Timestamp    int64
	Duration     *int
	TTFT         *int
	StopReason   string
	ErrorMessage string
	InputTokens  int
	OutputTokens int
	CacheRead    int
	CacheWrite   int
	TotalTokens  int
	CostInput    float64
	CostOutput   float64
	CostCacheRd  float64
	CostCacheWr  float64
	CostTotal    float64
}

type UserMessageStats struct {
	ID          int
	SessionFile string
	EntryID     string
	Folder      string
	Timestamp   int64
	Model       string
	Provider    string
	Chars       int
	Words       int
	Yelling     int
	Profanity   int
	Anguish     int
	Negation    int
	Repetition  int
	Blame       int
}

type AggregatedStats struct {
	TotalRequests       int
	SuccessfulRequests  int
	FailedRequests      int
	ErrorRate           float64
	TotalInputTokens    int
	TotalOutputTokens   int
	TotalCacheRead      int
	TotalCacheWrite     int
	CacheRate           float64
	TotalCost           float64
	TotalPremiumReqs    float64
	AvgDuration         *float64
	AvgTTFT             *float64
	AvgTokensPerSec     *float64
	FirstTimestamp      int64
	LastTimestamp       int64
}

type ModelStats struct {
	Model    string
	Provider string
	AggregatedStats
}

type FolderStats struct {
	Folder string
	AggregatedStats
}

type TimeSeriesPoint struct {
	Timestamp int64
	Requests  int
	Errors    int
	Tokens    int
	Cost      float64
}

type ModelTimeSeriesPoint struct {
	Timestamp int64
	Model     string
	Provider  string
	Requests  int
}

type ModelPerformancePoint struct {
	Timestamp       int64
	Model           string
	Provider        string
	Requests        int
	AvgTTFT         *float64
	AvgTokensPerSec *float64
}

type CostTimeSeriesPoint struct {
	Timestamp    int64
	Model        string
	Provider     string
	Cost         float64
	CostInput    float64
	CostOutput   float64
	CostCacheRd  float64
	CostCacheWr  float64
	Requests     int
}

type DashboardStats struct {
	Overall                AggregatedStats
	ByModel                []ModelStats
	ByFolder               []FolderStats
	TimeSeries             []TimeSeriesPoint
	ModelSeries            []ModelTimeSeriesPoint
	ModelPerformanceSeries []ModelPerformancePoint
	CostSeries             []CostTimeSeriesPoint
}

type BehaviorTimeSeriesPoint struct {
	Timestamp  int64
	Model      string
	Provider   string
	Messages   int
	Yelling    int
	Profanity  int
	Anguish    int
	Negation   int
	Repetition int
	Blame      int
	Chars      int
}

type BehaviorOverallStats struct {
	TotalMessages  int
	TotalYelling   int
	TotalProfanity int
	TotalAnguish   int
	TotalNegation  int
	TotalRepetition int
	TotalBlame     int
	TotalChars     int
	FirstTimestamp int64
	LastTimestamp  int64
}

type BehaviorModelStats struct {
	Model          string
	Provider       string
	TotalMessages  int
	TotalYelling   int
	TotalProfanity int
	TotalAnguish   int
	TotalNegation  int
	TotalRepetition int
	TotalBlame     int
	TotalChars     int
	LastTimestamp  int64
}

type BehaviorDashboardStats struct {
	Overall         BehaviorOverallStats
	ByModel         []BehaviorModelStats
	BehaviorSeries  []BehaviorTimeSeriesPoint
}

var (
	profanityWords = []string{
		"fuck", "fucks", "fucked", "fucking", "fuckin", "fucker", "fuckers",
		"fuckup", "fuckups", "fuckhead", "fuckheads", "fuckface", "fuckwit",
		"fuckwits", "fucktard", "fuckery", "fuckoff",
		"motherfucker", "motherfuckers", "motherfucking",
		"clusterfuck", "ratfuck", "unfuck",
		"fk", "fks", "fking", "fkin", "fker", "fck", "fcks", "fcking", "fckin", "fcker",
		"fuk", "fuking", "fukin",
		"eff", "effs", "effed", "effing",
		"frick", "fricks", "fricked", "fricking", "frickin",
		"freaking", "freakin", "freaked",
		"shit", "shits", "shat", "shitty", "shittier", "shittiest",
		"shite", "shites", "shited", "shitting", "shitter", "shitters",
		"shithead", "shitheads", "shitshow", "shitstorm", "shitstain",
		"shitfaced", "shitload", "shitbag", "shitcan", "shitcanned",
		"shitpost", "shitposting",
		"bullshit", "bullshits", "bullshitting", "bullshitter",
		"horseshit", "batshit", "dogshit", "dipshit", "jackshit",
		"dumbshit", "holyshit",
		"damn", "damns", "damned", "damning", "dammit",
		"goddamn", "goddamned", "goddamnit", "goddammit",
		"darn", "darns", "darned", "darnit",
		"dang", "danged", "dangit",
		"hell", "hells", "heck", "hecks", "heckin",
		"gosh", "blast", "blasted", "bloody", "bollocks", "bollox",
		"crap", "craps", "crappy", "crappier", "crappiest", "crapped", "crapping", "crapload", "crapshoot", "crapola",
		"piss", "pisses", "pissed", "pissing", "pisser", "pisspoor", "pisstake", "pisshead",
		"ass", "asses", "asshole", "assholes", "asshat", "asshats",
		"asswipe", "asswipes", "assclown", "assbag", "asskisser",
		"dumbass", "dumbasses", "jackass", "jackasses",
		"smartass", "smartasses", "badass", "badasses",
		"lazyass", "fatass", "hardass", "halfass", "halfassed",
		"arse", "arsed", "arsehole", "arseholes", "arsewipe",
		"bitch", "bitches", "bitched", "bitching", "bitchy", "bitchier", "bitchiest",
		"sonofabitch", "biatch", "biotch",
		"cunt", "cunts", "cunty", "cuntish",
		"twat", "twats", "twatty",
		"bastard", "bastards",
		"dick", "dicks", "dickhead", "dickheads", "dickish", "dickwad", "dickwads", "dickface", "dickbag",
		"prick", "pricks", "prickish",
		"cock", "cocks", "cocky", "cockier", "cockiest", "cockhead",
		"cockblock", "cocksucker", "cocksuckers",
		"knob", "knobhead", "knobheads", "knobend",
		"wanker", "wankers", "wankery",
		"tosser", "tossers",
		"jerkoff", "jerkoffs",
		"douche", "douches", "douchebag", "douchebags", "douchey",
		"scumbag", "scumbags", "scum",
		"sleazebag", "sleazeball", "slimeball",
		"lowlife", "lowlifes", "deadbeat",
		"idiot", "idiots", "idiotic", "idiocy",
		"stupid", "stupider", "stupidest", "stupidity",
		"moron", "morons", "moronic",
		"imbecile", "imbeciles",
		"retard", "retards", "retarded",
		"dumb", "dumber", "dumbest", "dumbo", "dummy", "dummies",
		"fool", "fools", "foolish", "foolery",
		"clown", "clowns", "clownish",
		"buffoon", "buffoons",
		"simpleton", "halfwit", "halfwits", "nitwit", "nitwits", "dimwit", "dimwits",
		"dolt", "dolts", "doltish",
		"knucklehead", "knuckleheads", "blockhead", "blockheads",
		"lamebrain", "airhead", "airheads", "scatterbrain",
		"numbnuts", "numbskull", "numpty", "numpties",
		"muppet", "muppets", "pillock", "pillocks", "plonker", "plonkers",
		"prat", "prats", "berk", "berks",
		"ninny", "ninnies", "dingbat", "dingbats",
		"putz", "putzes", "schmuck", "schmucks",
		"jerk", "jerks", "jerkface",
		"git", "gits", "sod", "sodding", "bugger", "buggered",
		"hate", "hated", "hates", "hating", "hateful",
		"suck", "sucks", "sucked", "sucking", "sucky", "suckage",
		"trash", "trashy", "trashed",
		"garbage", "crud", "crudded",
		"useless", "pointless", "horrible", "awful", "worthless", "ridiculous", "nonsense",
		"jesus", "christ", "jeez", "jeezus", "sheesh", "godsake",
		"wtf", "wth", "wtaf", "stfu", "gtfo", "omfg", "omg", "ffs", "jfc",
		"kys", "fml", "smh", "smdh", "smfh",
		"idgaf", "idfc", "lmfao", "fubar", "snafu",
		"ugh", "ughh", "ughhh", "urgh",
		"argh", "arghh", "arghhh", "arrgh",
		"blah", "bleh", "meh",
		"yikes", "yeesh", "oof",
		"gah", "gahh", "grr", "grrr", "grrrr",
	}
	profanityRE     *regexp.Regexp
	dramaRE         *regexp.Regexp
	anguishRE       *regexp.Regexp
	dudeRE          *regexp.Regexp
	ellipsisRE      *regexp.Regexp
	negationLeadRE  *regexp.Regexp
	negationPhraseRE *regexp.Regexp
	repetitionRecallRE *regexp.Regexp
	repetitionStillRE  *regexp.Regexp
	blameYouRE      *regexp.Regexp
	blameStopRE     *regexp.Regexp
	fenceCodeRE     *regexp.Regexp
	xmlPairRE       *regexp.Regexp
	xmlBareRE       *regexp.Regexp
	inlineCodeRE    *regexp.Regexp
	urlRE           *regexp.Regexp
	fileMentionRE   *regexp.Regexp
	quoteLineRE     *regexp.Regexp
	imageMarkerRE   *regexp.Regexp
	ansiEscapeRE    *regexp.Regexp
	sentenceRE      *regexp.Regexp
	wordRE          *regexp.Regexp
)

func init() {
	profanityRE = regexp.MustCompile(`(?i)\b(?:` + strings.Join(profanityWords, "|") + `)\b`)
	dramaRE = regexp.MustCompile(`[!?][!?1]{2,}`)
	anguishRE = regexp.MustCompile(`(?i)\b(?:no{3,}|a+h{2,}|u+g+h{2,}|a+r+g+h+|st+o{3,}p+|w+h+y{3,}|f+u{3,}c*k*|wtf{3,}|o+m+g{2,}|ye+s{3,}|g+o+d{3,}|br+u+h{2,})\b`)
	dudeRE = regexp.MustCompile(`(?i)\bdude\b`)
	ellipsisRE = regexp.MustCompile(`\.{2,}`)
	negationLeadRE = regexp.MustCompile(`(?i)^[ \t]*(?:no|nope|nah|nvm|wrong|incorrect)\b`)
	negationPhraseRE = regexp.MustCompile(`(?i)\b(?:that(?:'|` + rsquo + `)s\s+not\s+(?:what|right|it)|not\s+what\s+i\s+(?:meant|asked|said|wanted))\b`)
	repetitionRecallRE = regexp.MustCompile(`(?i)\b(?:(?:like|as)\s+i\s+(?:said|told\s+you|asked)|i\s+(?:meant|said|told\s+you|asked\s+you|already\s+(?:said|told|did|asked|wrote)))\b`)
	repetitionStillRE = regexp.MustCompile(`(?i)\bstill\s+(?:doesn(?:'|` + rsquo + `)?t|doesnt|isn(?:'|` + rsquo + `)?t|isnt|not|broken|wrong|fails|failing|the\s+same|same)\b`)
	blameYouRE = regexp.MustCompile(`(?i)\byou\s+(?:didn(?:'|` + rsquo + `)?t|did\s+not|broke|missed|forgot|keep|always|never|still|ignored)\b`)
	blameStopRE = regexp.MustCompile(`(?i)(?:^|[.!?\n])\s*stop\s+\w+ing\b`)
	fenceCodeRE = regexp.MustCompile("```[\\s\\S]*?```")
	xmlPairRE = regexp.MustCompile(`<[A-Za-z][\w-]*\b[^>]*>[\s\S]*?<\/[A-Za-z][\w-]*\s*>`)
	xmlBareRE = regexp.MustCompile(`<\/?[A-Za-z][\w-]*\b[^>]*\/?>`)
	inlineCodeRE = regexp.MustCompile("`[^`\n]*`")
	urlRE = regexp.MustCompile(`\bhttps?://\S+`)
	fileMentionRE = regexp.MustCompile(`(^|\s)@[\w./-]+`)
	quoteLineRE = regexp.MustCompile(`(?m)^[ \t]*>.*$`)
	imageMarkerRE = regexp.MustCompile(`\[Image #\d+\]`)
	ansiEscapeRE = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	sentenceRE = regexp.MustCompile(`[^.!?\n]+`)
	wordRE = regexp.MustCompile(`\S+`)
}

func countMatches(text string, re *regexp.Regexp) int {
	n := 0
	for _, match := range re.FindAllString(text, -1) {
		_ = match
		n++
	}
	return n
}

func countLetters(text string) int {
	n := 0
	for _, r := range text {
		if unicode.IsLetter(r) {
			n++
		}
	}
	return n
}

func countUpperLetters(text string) int {
	n := 0
	for _, r := range text {
		if unicode.IsUpper(r) {
			n++
		}
	}
	return n
}

func countYellingSentences(text string) int {
	count := 0
	for _, match := range sentenceRE.FindAllString(text, -1) {
		letters := countLetters(match)
		if letters >= 4 {
			upper := countUpperLetters(match)
			if float64(upper)/float64(letters) > 0.5 {
				count++
			}
		}
	}
	return count
}

func stripStructuredContent(text string) string {
	s := text
	s = fenceCodeRE.ReplaceAllString(s, "\n")
	s = xmlPairRE.ReplaceAllString(s, "\n")
	s = xmlBareRE.ReplaceAllString(s, " ")
	s = inlineCodeRE.ReplaceAllString(s, " ")
	s = urlRE.ReplaceAllString(s, " ")
	s = fileMentionRE.ReplaceAllString(s, "$1 ")
	s = quoteLineRE.ReplaceAllString(s, "")
	s = imageMarkerRE.ReplaceAllString(s, " ")
	s = ansiEscapeRE.ReplaceAllString(s, "")
	return s
}

func countNonEmptyLines(text string) int {
	count := 0
	for _, line := range strings.Split(text, "\n") {
		if len(strings.TrimSpace(line)) > 0 {
			count++
		}
	}
	return count
}

func ComputeUserMetrics(text string) UserMessageMetrics {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return UserMessageMetrics{}
	}

	chars := len([]rune(trimmed))
	words := countMatches(trimmed, wordRE)

	prose := strings.TrimSpace(stripStructuredContent(trimmed))
	if prose == "" || countNonEmptyLines(prose) >= 3 {
		return UserMessageMetrics{
			Chars: chars,
			Words: words,
		}
	}

	anguish := countMatches(prose, dramaRE) +
		countMatches(prose, anguishRE) +
		countMatches(prose, dudeRE) +
		countMatches(prose, ellipsisRE)

	negation := countMatches(prose, negationLeadRE) + countMatches(prose, negationPhraseRE)
	repetition := countMatches(prose, repetitionRecallRE) + countMatches(prose, repetitionStillRE)
	blame := countMatches(prose, blameYouRE) + countMatches(prose, blameStopRE)

	return UserMessageMetrics{
		Chars:      chars,
		Words:      words,
		Yelling:    countYellingSentences(prose),
		Profanity:  countMatches(prose, profanityRE),
		Anguish:    anguish,
		Negation:   negation,
		Repetition: repetition,
		Blame:      blame,
	}
}
