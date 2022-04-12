# simplemockserver

The intention behind this library was to create a simple REST Server which use predefined endpoints
with the possibility to return different responses depending on filter queries on URL parameters 
or json body content for (POST, PUT)

## Getting started

```go
import (
    "github.com/stretchr/testify/require"
    "github.com/daolis/simplemockserver"
)

func TestNewMockServer(t *testing.T) {
	// use default configuration with config file 'testfiles/mock.json' and random free port.
	mockServer, err := NewMockServer() 
	require.NoError(t, err)
	defer mockServer.Stop()
    
	// get the mock server URL (e.g. http://localhost:12345) 
	mockURL := mockServer.GetURL()
	_ = mockURL
	
	// use custom file and fixed port 112233
	mockServer, err := NewMockServer(WithFile("customFile.json"), WithFixedPort(112233))
	require.NoError(t, err)
}
```

## Configure endpoints

**Example configuration**
```json
{
    "/endpoint2": {
        "method": "POST",
        "responses": [
            {
                "requestQuery": {
                    "body": "name = 'John'"
                },
                "response": {
                    "status": 200,
                    "body": {
                        "testKey1": "testValue1",
                        "testKey2": "testValue2"
                    }
                }
            },
            {
                "requestQuery": {
                    "body": "name = 'Doe'"
                },
                "response": {
                    "status": 200,
                    "body": {
                        "name": "Doe",
                        "age": 125,
                        "other": "test"
                    }
                }
            }
        ]
    }
}
```

Here we configure one endpoint `/endpoint2` white method `POST` which can return two different responses.
The `requestQuery` parameter defines which response will be sent to the client.

1. `"body": "name = 'John'"`: The body has to be JSON will be chosen if the body contains a attribute `name` with the value `John`.\
   e.g.
   ```json
   {
       "test": "lala",
       "name": "John"
   }
   ```
2. same with name `Doe`

The response with the first matching query will be returned! If no response is matching it will return a `404 NOT FOUND`.
