To fit longer context, you can use KV cache quantization to quantize the K and V caches to lower bits. This can also increase generation speed due to reduced RAM / VRAM data movement. The allowed options for K quantization (default is f16) include the below.

--cache-type-k f32, f16, bf16, q8_0, q4_0, q4_1, iq4_nl, q5_0, q5_1 

You should use the _1 variants for somewhat increased accuracy, albeit it's slightly slower. For eg q4_1, q5_1 

You can also quantize the V cache, but you will need to compile llama.cpp with Flash Attention support via -DGGML_CUDA_FA_ALL_QUANTS=ON, and use --flash-attn to enable it.

We also uploaded 1 million context length GGUFs via YaRN scaling here.
