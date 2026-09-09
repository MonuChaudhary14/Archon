import os
from typing import Dict, List, Tuple
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.sdk.resources import Resource
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator

_tracer = None

def init_telemetry(service_name: str = None, otlp_endpoint: str = None):
    global _tracer
    if _tracer is not None:
        return _tracer

    service_name = service_name or os.getenv("OTEL_SERVICE_NAME", "archon-ai-agent")
    otlp_endpoint = otlp_endpoint or os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "jaeger:4317")

    resource = Resource.create({"service.name": service_name})
    provider = TracerProvider(resource=resource)

    try:
        from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
        exporter = OTLPSpanExporter(endpoint=otlp_endpoint, insecure=True)
        processor = BatchSpanProcessor(exporter)
        provider.add_span_processor(processor)
    except Exception as e:
        print(f"OTLP trace exporter init warning: {e}")

    trace.set_tracer_provider(provider)
    _tracer = trace.get_tracer(service_name)
    return _tracer

def get_tracer():
    global _tracer
    if _tracer is None:
        return init_telemetry()
    return _tracer

def extract_trace_context_from_headers(headers: List[Tuple[str, bytes]]):
    if not headers:
        return None
    carrier: Dict[str, str] = {}
    for key, value in headers:
        try:
            carrier[key.lower()] = value.decode("utf-8") if isinstance(value, bytes) else str(value)
        except Exception:
            pass
    propagator = TraceContextTextMapPropagator()
    return propagator.extract(carrier=carrier)

def inject_trace_context_to_headers(context=None) -> List[Tuple[str, bytes]]:
    carrier: Dict[str, str] = {}
    propagator = TraceContextTextMapPropagator()
    propagator.inject(carrier=carrier, context=context)
    return [(k, v.encode("utf-8")) for k, v in carrier.items()]
