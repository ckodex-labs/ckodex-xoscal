using System;
using System.Buffers.Binary;
using System.Collections.Generic;
using System.Net;
using System.Net.Http;
using System.Net.Http.Headers;
using System.Threading;
using System.Threading.Tasks;
using Google.Protobuf;
using Grpc.Net.Client;
using Oscal.Services.V1;

var message = new ListCatalogsRequest { PageSize = 17 };
if (ListCatalogsRequest.Parser.ParseFrom(message.ToByteArray()).PageSize != 17)
    throw new Exception("protobuf roundtrip");
if (GovernanceServiceReflection.Descriptor.Services.Count == 0 ||
    TransparencyExchangeServiceReflection.Descriptor.Services.Count == 0 ||
    TransparencyGraphServiceReflection.Descriptor.Services.Count == 0)
    throw new Exception("service metadata missing");
var handler = new LocalGrpcHandler();
using var channel = GrpcChannel.ForAddress("https://sdk-consumer.invalid", new GrpcChannelOptions { HttpHandler = handler });
var response = await new OscalService.OscalServiceClient(channel).ListCatalogsAsync(new ListCatalogsRequest { PageSize = 17 });
if (response.NextPageToken != "consumer") throw new Exception("gRPC response marshalling");
await new GovernanceService.GovernanceServiceClient(channel).ListConflictsAsync(new ListConflictsRequest());
await new TransparencyExchangeService.TransparencyExchangeServiceClient(channel).ListClaimsAsync(new ListClaimsRequest());
await new TransparencyGraphService.TransparencyGraphServiceClient(channel).ListNodesAsync(new ListNodesRequest());
if (handler.Routes.Count != 4) throw new Exception("four service routes not exercised");
Console.WriteLine("C# SDK installed package, messages, and four gRPC client wire routes passed (local HTTP handler)");

sealed class LocalGrpcHandler : HttpMessageHandler
{
    public HashSet<string> Routes { get; } = new HashSet<string>();
    protected override async Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
    {
        if (request.Version.Major != 2 || request.Content.Headers.ContentType.MediaType != "application/grpc")
            throw new Exception("expected HTTP/2 gRPC request");
        var bytes = await request.Content.ReadAsByteArrayAsync(cancellationToken);
        if (bytes.Length < 5 || bytes[0] != 0 || BinaryPrimitives.ReadInt32BigEndian(bytes.AsSpan(1, 4)) != bytes.Length - 5)
            throw new Exception("gRPC request framing");
        var route = request.RequestUri.AbsolutePath;
        IMessage reply;
        switch (route)
        {
            case "/oscal.services.v1.OscalService/ListCatalogs":
                if (ListCatalogsRequest.Parser.ParseFrom(bytes.AsSpan(5).ToArray()).PageSize != 17)
                    throw new Exception("gRPC request marshalling");
                reply = new ListCatalogsResponse { NextPageToken = "consumer" };
                break;
            case "/oscal.services.v1.GovernanceService/ListConflicts":
                ListConflictsRequest.Parser.ParseFrom(bytes.AsSpan(5).ToArray());
                reply = new ListConflictsResponse(); break;
            case "/oscal.services.v1.TransparencyExchangeService/ListClaims":
                ListClaimsRequest.Parser.ParseFrom(bytes.AsSpan(5).ToArray());
                reply = new ListClaimsResponse(); break;
            case "/oscal.services.v1.TransparencyGraphService/ListNodes":
                ListNodesRequest.Parser.ParseFrom(bytes.AsSpan(5).ToArray());
                reply = new ListNodesResponse(); break;
            default: throw new Exception("unexpected gRPC route: " + route);
        }
        Routes.Add(route);
        var payload = reply.ToByteArray();
        var frame = new byte[payload.Length + 5];
        BinaryPrimitives.WriteInt32BigEndian(frame.AsSpan(1, 4), payload.Length);
        payload.CopyTo(frame, 5);
        var result = new HttpResponseMessage(HttpStatusCode.OK) { Version = new Version(2, 0), Content = new ByteArrayContent(frame) };
        result.Content.Headers.ContentType = new MediaTypeHeaderValue("application/grpc");
        result.TrailingHeaders.Add("grpc-status", "0");
        return result;
    }
}
