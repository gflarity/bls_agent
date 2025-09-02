package arxiv

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gflarity/bls_agent/pkg/llm"
	"go.temporal.io/sdk/workflow"
)

type PaperOfTheDayWorkflowParams struct {
	Date time.Time
	// OpenAI configuration
	OpenAIAPIKey  string `json:"openai_api_key"`
	OpenAIBaseURL string `json:"openai_base_url"`
	OpenAIModel   string `json:"openai_model"`
}

func PaperOfTheDayWorkflow(ctx workflow.Context, params PaperOfTheDayWorkflowParams) ([]string, error) {

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 60 * time.Second,
	})

	// fetch arxiv ids for the date
	var arxivIds []string
	err := workflow.ExecuteActivity(ctx, GetArxivIdsForDateActivity, params.Date).Get(ctx, &arxivIds)
	if err != nil {
		return nil, fmt.Errorf("failed to get arxiv ids: %w", err)
	}
	//   loop through the ids
	var ids []string
	for _, arxivId := range arxivIds {

		// setup timer to avoid rate limiting, by waiting 1 second between requests (if required)
		timer := workflow.NewTimer(ctx, 1*time.Second)

		//   get the abstract
		var abs string
		err := workflow.ExecuteActivity(ctx, GetArxivAbstractActivity, arxivId).Get(ctx, &abs)
		if err != nil {
			return nil, fmt.Errorf("failed to get arxiv abstract: %w", err)
		}

		//   filter unwanted papers based on abstract
		var keeper struct {
			Keep bool `json:"keep" jsonschema:"description=Whether the paper should be kept,title=Keep Paper"`
		}

		schema, err := llm.GenerateSchema(keeper)
		if err != nil {
			return nil, fmt.Errorf("failed to generate schema from type: %w", err)
		}

		// TODO just use a param struct
		sys := fmt.Sprintf("You are an expert AI Research Analyst. Here's the json schema you need to adhere to: <schema>%s</schema>", schema)
		user := fmt.Sprintf(`
Your task is to filter academic abstracts to identify groundbreaking research in AI efficiency.

Primary Directive:
Your sole focus is to identify papers that introduce novel methods, algorithms, architectures, or hardware/software co-design techniques specifically aimed at improving the performance-per-dollar of AI/ML/LLM training or inference. The contribution must be a direct improvement to the AI/ML model or system itself, not an application of AI that saves money in another domain.

Inclusion Criteria (Answer true):
The abstract must describe a new technique related to:

Model optimization (e.g., quantization, pruning, knowledge distillation, sparsity).

Algorithmic efficiency (e.g., faster attention mechanisms, optimized training steps).

System-level improvements (e.g., compiler optimizations for ML workloads, efficient data parallelism strategies).

Specialized hardware for accelerating AI tasks.

Exclusion Criteria (Answer false):
The abstract should be rejected if it:

Simply uses an existing ML/LLM model to solve a problem more efficiently in another field (e.g., finance, logistics, biology).

Discusses the economic or social impact of AI costs without proposing a technical solution.

Describes improvements to a data pipeline or MLOps process that do not change the core training/inference efficiency.

Example 1 (Correctly identify as true)

Abstract: "We introduce 'Sparse-Quant,' a novel post-training quantization algorithm that applies structured pruning to large language models. Our method reduces the memory footprint by 60% and increases inference throughput by 2.5x on standard benchmarks with less than a 1% drop in accuracy. This enables the deployment of billion-parameter models on commodity hardware, significantly reducing operational costs."

Your Reasoning: This abstract introduces a new algorithm (Sparse-Quant) that directly improves inference throughput and reduces memory, which are core metrics for performance-per-dollar in AI systems. The answer is true.

Example 2 (Correctly identify as false)

Abstract: "This paper demonstrates the application of a transformer-based LLM to optimize global supply chain routing. By analyzing historical shipping data, our model generates routes that reduce fuel consumption and operational costs by 15% compared to traditional methods. Our findings show that leveraging AI can create more sustainable and cost-effective logistics networks."

Your Reasoning: This abstract uses an LLM to solve a logistics problem. The innovation is in the application of AI, not in making the LLM itself more efficient. The cost savings are in logistics, not in the model's training or inference. The answer is false.

Task:
Analyze the following abstract based on the directive and criteria above. Does this abstract focus on a new technique to improve the efficiency or cost-effectiveness of ML/LLM training or inference?

Respond with a JSON object containing a single key is_relevant with a boolean value (true or false). Do not add any other text or explanation.

Abstract: %s`, abs)

		model := "deepseek/deepseek-r1-0528"
		// TODO need to implement better reasoning support for DS V3.1, in the mean time just use DSR1
		var res string
		err = workflow.ExecuteActivity(ctx, CompleteWithSchemaActivity, params.OpenAIAPIKey, params.OpenAIBaseURL, schema, sys, user, model).Get(ctx, &res)
		if err != nil {
			return nil, fmt.Errorf("failed to complete with schema: %w", err)
		}

		err = json.Unmarshal([]byte(res), &keeper)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal response into keeper: %w", err)
		}

		if keeper.Keep {
			ids = append(ids, arxivId)
		}

		// ensure we wait at least 1 second between requests, if it's already been 1 second this won't block
		timer.Get(ctx, nil)
	}

	return ids, nil

	//   if this paper looks promising, fetch the full text

	//   compare this paper's text to current leaders text and
	//   pick one

	// post the leader to twitter

}
