from datetime import timedelta
from typing import Dict, Any
from temporalio import workflow
from temporalio.common import RetryPolicy

with workflow.unsafe.imports_passed_through():
    from app.workflows.activities import EvaluationActivities
    from app.workflows.contracts import EvaluationWorkflowInput

@workflow.defn
class SystemDesignEvaluationWorkflow:
    @workflow.run
    async def run(self, input_data: EvaluationWorkflowInput) -> Dict[str, Any]:
        session_id = input_data.session_id

        standard_retry = RetryPolicy(
            initial_interval=timedelta(seconds=1),
            backoff_coefficient=2.0,
            maximum_interval=timedelta(seconds=10),
            maximum_attempts=3
        )

        llm_retry = RetryPolicy(
            initial_interval=timedelta(seconds=2),
            backoff_coefficient=2.0,
            maximum_interval=timedelta(seconds=15),
            maximum_attempts=3
        )

        try:
            context = await workflow.execute_activity_method(
                EvaluationActivities.fetch_interview_context,
                session_id,
                start_to_close_timeout=timedelta(seconds=30),
                retry_policy=standard_retry
            )

            expected_topics = context.get("expected_topics", [])
            knowledge_rubrics = []
            if expected_topics:
                knowledge_rubrics = await workflow.execute_activity_method(
                    EvaluationActivities.search_knowledge_rubrics,
                    expected_topics,
                    start_to_close_timeout=timedelta(seconds=30),
                    retry_policy=standard_retry
                )

            evaluation_payload = {
                "context": context,
                "knowledge_rubrics": knowledge_rubrics
            }

            evaluation_result = await workflow.execute_activity_method(
                EvaluationActivities.execute_llm_evaluation,
                evaluation_payload,
                start_to_close_timeout=timedelta(seconds=120),
                retry_policy=llm_retry
            )

            overall_score = evaluation_result.get("overall_score", 0)
            persist_payload = {
                "session_id": session_id,
                "score": overall_score,
                "feedback": evaluation_result
            }

            await workflow.execute_activity_method(
                EvaluationActivities.persist_evaluation_report,
                persist_payload,
                start_to_close_timeout=timedelta(seconds=30),
                retry_policy=standard_retry
            )

            return {
                "status": "COMPLETED",
                "session_id": session_id,
                "overall_score": overall_score,
                "feedback": evaluation_result
            }

        except Exception as exc:
            error_message = str(exc)
            compensation_payload = {
                "session_id": session_id,
                "error_message": error_message
            }
            await workflow.execute_activity_method(
                EvaluationActivities.compensate_evaluation_failure,
                compensation_payload,
                start_to_close_timeout=timedelta(seconds=30),
                retry_policy=standard_retry
            )
            return {
                "status": "FAILED",
                "session_id": session_id,
                "error": error_message
            }
