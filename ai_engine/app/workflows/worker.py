import asyncio
from temporalio.client import Client
from temporalio.worker import Worker
from app.core.config import settings
from app.workflows.evaluation_workflow import SystemDesignEvaluationWorkflow
from app.workflows.activities import EvaluationActivities

EVALUATION_TASK_QUEUE = "archon-interview-eval-queue"

class TemporalEvaluationWorker:
    def __init__(self, activities: EvaluationActivities | None = None):
        self.activities = activities or EvaluationActivities()
        self.worker: Worker | None = None
        self.client: Client | None = None

    async def start(self):
        try:
            self.client = await Client.connect(
                settings.TEMPORAL_HOST,
                namespace=settings.TEMPORAL_NAMESPACE
            )
            self.worker = Worker(
                self.client,
                task_queue=EVALUATION_TASK_QUEUE,
                workflows=[SystemDesignEvaluationWorkflow],
                activities=[
                    self.activities.fetch_interview_context,
                    self.activities.search_knowledge_rubrics,
                    self.activities.execute_llm_evaluation,
                    self.activities.persist_evaluation_report,
                    self.activities.compensate_evaluation_failure,
                ],
            )
            await self.worker.run()
        except asyncio.CancelledError:
            pass
        except Exception as e:
            print(f"Temporal worker encountered error: {e}")

    async def execute_evaluation_workflow(self, session_id: str):
        if not self.client:
            self.client = await Client.connect(
                settings.TEMPORAL_HOST,
                namespace=settings.TEMPORAL_NAMESPACE
            )
        from app.workflows.contracts import EvaluationWorkflowInput
        handle = await self.client.start_workflow(
            SystemDesignEvaluationWorkflow.run,
            EvaluationWorkflowInput(session_id=session_id),
            id=f"interview-eval-{session_id}",
            task_queue=EVALUATION_TASK_QUEUE,
        )
        return handle
