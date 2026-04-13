# Notes, Architectural decisions and the Building Process

## The basic User Interface

I wanted to know a little bit more about what problem we were trying to solve before jumping too deep into the weeds.
The first thing I did was to define the interface of the crawler. I wanted to be able to run it from the command line
and just wanted basic way to test as I went so I opted for just two parameters the seeed-url and the depth.
In Hindsight this was a smart decision as the sample website I was crawling turned out to be much larger than I
expected.

## Defining Children and cleaning the URLs

It didn't seem obvious exactly how one might uncontroversially define the children of a page. I decided to go with the
most basic definition of children as the in-domain links found on a page. This is a simple definition that is easy to
understand and implement, but it does have some limitations. For example, it doesn't account for links that are hidden
behind JavaScript or links that are only accessible through forms.
But I created an interface `ChildSource` which allows for easy extension in the future if we wanted to add more complex
definitions of children. It also enables easy testing of the crawler by allowing us to statically define the child
source.

I've worked on a couple of scrapers before and come up against http vs https, trailing slashes, ports and anchors which
can impede the crawler from correctly identifying in-domain links.

## The first implementation

I decided to implement a breadth-first search (BFS) crawler as it is a simple and effective way to traverse a graph.
It also allows us to easily limit the depth of the crawl which I thought from the get-go was a crucial parameter for us
to observe.
Fo simplicity I implemented the crawler in a single-threaded manner as it is easier to reason about and debug.

## The internal data structures

I use two maps and a queue to keep track of the internal state. 
One maps allow us to quickly check if a page has already been visited to prevent infinite loops, the other keeps track of the results and any errors. 
Finally the queue allows us to easily implement the BFS traversal.
Additionally visitOrder allows for deterministic output which is important for testing and debugging.

```
	visited := map[string]bool{}
	results := map[string]PageResult{}
	visitOrder := make([]string, 0)
	queue := []queueItem{{URL: normalizedSeed, Depth: 0}}
```

## Testing the solution

Even testing with a depth of 3, on the trial website, the crawler seemed to be stuck but I had no easy way to tell until
I added the `--debug` flag which prints out the URLs as they are being visited.
This was a crucial addition as it allowed me to see that the crawler was indeed making progress and not stuck in an
infinite loop, the issue was that the graph was very broad and all the I/O operations were taking a long time to
complete.

## Introducing concurrency

To speed up the crawling process, I decided to implement a concurrent version of the crawler using goroutines and
channels. This allows us to fetch multiple pages in parallel, which can significantly reduce the time it takes to crawl
a large website.
However to test that concurrency was working correctly I had to add a `--runner` flag which allows us to select between
the single-threaded and concurrent implementations of the crawler proviiding a way to easily compare the two and ensure
that they are producing the same results.
Additionally I added a `--summary` flag which prints out aggregate crawl metrics by depth instead of listing every
page/link. This allows us to easily see the overall structure of the crawl and timimng info to easily allow the
comparison between the single and multi-threaded versions.

I achieved the following results from the command

`go run . https://crawlme.monzo.com/  --summary  --runner=multi`

using a maximum of 1000 concurrent go routines.

```
Total pages found: 42011
Depth 0: found=1 scraped=1 avg_scrape_time=56.452ms
Depth 1: found=10 scraped=10 avg_scrape_time=14.707ms
Depth 2: found=2 scraped=2 avg_scrape_time=13.867ms
Depth 3: found=44 scraped=44 avg_scrape_time=23.046ms
...
...
Depth 504: found=84 scraped=84 avg_scrape_time=59.885ms
Depth 505: found=83 scraped=83 avg_scrape_time=58.733ms
Depth 506: found=60 scraped=60 avg_scrape_time=57.883ms
Overall totals:
found=42011 scraped=42011 total_run_time=29.245436s avg_scrape_time=59.738ms
```

I figured a runtime of just under 30 seconds for crawling over 42,000 pages was pretty good and a significant
improvement over the single-threaded version which would take around 40 minutes to complete the same crawl.

