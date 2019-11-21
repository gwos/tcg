/*
typedef struct _string_transit_TypedValue_Pair_List_ {
    size_t count;
    string_transit_TypedValue_Pair *items;
} string_transit_TypedValue_Pair_List;
*/
string_transit_TypedValue_Pair_List *JSON_as_string_transit_TypedValue_Pair_List(json_t *json) {
    string_transit_TypedValue_Pair_List *Pair_List = (string_transit_TypedValue_Pair_List *)malloc(sizeof(string_transit_TypedValue_Pair_List));
    if (!Pair_List) {
	// FIX MAJOR:  invoke proper logging for error conditions
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_string_transit_TypedValue_Pair_List, %s\n", "malloc failed");
    } else {
	int failed = 0;
	Pair_List->count = json_object_size(json);
	Pair_List->items = (string_transit_TypedValue_Pair *)malloc(Pair_List->count * sizeof(string_transit_TypedValue_Pair));
	const char *key;
	json_t *value;
	size_t i = 0;
	json_object_foreach(json, key, value) {
	    // Here we throw away constness as far as the compiler is concerned, but by convention
	    // the calling code will never alter the key, so that won't matter.
	    Pair_List->items[i].key = (char *) key;
	    transit_TypedValue *TypedValue_ptr = JSON_as_transit_TypedValue(value);
	    if (TypedValue_ptr == NULL) {
		fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_string_transit_TypedValue_Pair_List, %s\n", "TypedValue_ptr is NULL");
		failed = 1;
	    } else {
		Pair_List->items[i].value = *TypedValue_ptr;
	    }
	    fprintf(stderr, "processed key %s\n", key);
	    ++i;
	}
	if (failed) {
	    free(Pair_List);
	    Pair_List = NULL;
	}
    }
    return Pair_List;
}
