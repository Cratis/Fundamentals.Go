// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.
// Boundary witness only; checked outside every observation catch because TypeConversion catches errors.
namespace Cratis.Json;
public static class Globals
{
    public static int AccessCount { get; private set; }
    public static System.Text.Json.JsonSerializerOptions JsonSerializerOptions
    {
        get
        {
            AccessCount++;
            throw new InvalidOperationException("Probe reached unimplemented global serializer configuration");
        }
    }
}
