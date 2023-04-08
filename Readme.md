# httpreadat: Go http range requests via an io.ReaderAt

httpreadat is a library for making http range requests exposed as a normal io.ReaderAt
interface.

This is especially useful for interacting with large files stored in object stores like S3,
where you only need to read certain parts of the file.

## Example

```
	rr := New(url)
	rr.ReadAt(buf, offset)
```

## Options

### Request manipulation

You may need more control over the request beyond simply setting the url. For those cases
you can optionally provide a custom `http.RoundTripper`. The RoundTripper has full access
to the request to make any changes necessary before sending the request out.

### Caching

Caching is often useful, especially when used with code that is not optimized for doing
reads over the network. `httpreadat` provides an optional interface `CacheHandler` that
you can implement for your own caching strategy.

The cache handler can make requests that are different than what ReadAt was called with.
This allows for things like fetching a larger amount of data if the caller makes lots
of small sequential reads.

There is an example `CacheHandler` in `diskcache` that fetches pages at a time and
caches to a local os.File.

## S3

The easiest way to use this with S3 is to make a presigned url for the S3 object and
pass that url to `New()`:

```
	req, _ := s3svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	})

	url, err := req.Presign(1 * time.Hour)
	if err != nil {
		return err
	}

	r := httpreadat.New(url)
```
