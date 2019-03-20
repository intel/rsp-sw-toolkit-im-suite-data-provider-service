# Expect for Go

Expect is a simple testing helper library to make Go's test expectations more obvious.

Add to your project with: 
```bash
> govendor fetch github.impcloud.net/Responsive-Retail-Inventory/expect
```

## Usage and Examples

Wrap a `*testing.T` instance with a `*expect.TWrapper`. You can use it just as
you would the normal testing instance, or us it to help make expectations clear:

```go
func TestSomething(t *testing.T) {
    // wrap t to use it
    w := expect.WrapT(t)
    
    // normal testing.T operations work as usual
    w.Log("some log message") 
    w.Error("error message")    
    
    // but now expectations are more obvious
    w.ShouldNotBeNil(resp)
    w.ShouldBeEqual(resp.StatusCode, 200)
    content := w.ShouldHaveResult(ioutil.ReadAll(resp.Body)).([]byte)
    w.ShouldBeEqual(content, []byte("hello"))
}
```

### Error Handling
When something goes wrong, `expect` shows the failed expectation, which you can
augment with a message. You can also specify clearly whether tests should continue:

```go
    w.As("proxy state").ShouldBeFalse(fc.IsProxied())
    w.StopOnMismatch().As("download function").ShouldBeNil(fc.downloadFunc)
```

Shows the following:
```
--- FAIL: TestProxyThroughCloudConnector (0.00s)
    filecache_test.go:68: Failure for 'proxy state': false != true (types bool, bool)
    filecache_test.go:69: Failure for 'download function': true != false (types bool, bool)
```

`As` takes an interface, printed with `%+v`, so it's useful for more than just strings.

### Functions
Expect makes it easier to handle functions that return errors and other types:

```go
    // should succeed makes sure that a function's returned error is nil
    w.ShouldSucceed(req.Reply(req.Type == "shell", nil))

    // if a function has multiple returns, you can check the error directly
    t1, t2, t3, err := threeTs()
    w.ShouldSucceed(err)
    
    // for the more common case of 1 return and 1 error, simply type-assert the result 
    resp := w.ShouldHaveResult(download("http://someurl.com/")).(*http.Response)
    

    // simply type-assert an expected result for functions that return (<T>, error)
    testfile := w.ShouldHaveResult(ioutil.TempFile("", "")).(*os.File)
	
    // ShouldSucceedLater returns a function that calls & checks a function later on;
    // this is most useful for functions that are deferred.
    defer w.ShouldSucceedLater(func() error { return os.Remove(testfile.Name()) })
    defer w.ShouldSucceedLater(testfile.Close())
    testfileContent := w.ShouldHaveResult(ioutil.ReadFile(testfile.Name())).([]byte)
    w.ShouldBeEqual(testfileContent, []byte("hello"))
    w.ShouldBeEqual(testfileContent, content)
```

## List of Functions

- WrapT(t *testing.T) *TWrapper
- (t *TWrapper) As(d interface{}) *TWrapper
- (t *TWrapper) StopOnMismatch() *TWrapper
- (t *TWrapper) ContinueOnMismatch() *TWrapper
- (t *TWrapper) ShouldBeEqual(x, y interface{})
- (t *TWrapper) ShouldNotBeEqual(x, y interface{})
- (t *TWrapper) ShouldBeSameType(x, y interface{})
- (t *TWrapper) ShouldBeFalse(v interface{})
- (t *TWrapper) ShouldBeTrue(v interface{})
- (t *TWrapper) ShouldNotBeNil(v interface{})
- (t *TWrapper) ShouldBeNil(v interface{})
- (t *TWrapper) ShouldBeEmptyStr(v interface{})
- (t *TWrapper) ShouldNotBeEmptyStr(v interface{})
- (t *TWrapper) ShouldContainStr(x, y string)
- (t *TWrapper) ShouldFail(err error) error
- (t *TWrapper) ShouldSucceed(err error)
- (t *TWrapper) ShouldSucceedLater(f func() error) func() error
- (t *TWrapper) ShouldHaveError(result interface{}, err error) error
- (t *TWrapper) ShouldHaveResult(result interface{}, err error) interface{}
- (t *TWrapper) ShouldHaveLength(itemWithLen interface{}, length int)
